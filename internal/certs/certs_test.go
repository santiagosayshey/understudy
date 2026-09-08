package certs

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "certs")
	if err := Generate(dir, "metadata-static.plex.tv"); err != nil {
		t.Fatal(err)
	}
	for _, f := range Files {
		st, err := os.Stat(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(f, ".key") && st.Mode().Perm() != 0o600 {
			t.Errorf("%s should be private, got %v", f, st.Mode().Perm())
		}
	}
	// The leaf must verify against the authority for the hostname, the way
	// Plex will check it.
	caPEM, _ := os.ReadFile(filepath.Join(dir, "ca.crt"))
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("ca.crt is not a certificate")
	}
	pair, err := tls.LoadX509KeyPair(filepath.Join(dir, "leaf.crt"), filepath.Join(dir, "leaf.key"))
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := x509.ParseCertificate(pair.Certificate[0])
	if _, err := leaf.Verify(x509.VerifyOptions{DNSName: "metadata-static.plex.tv", Roots: pool}); err != nil {
		t.Fatalf("leaf does not verify: %v", err)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{DNSName: "plex.tv", Roots: pool}); err == nil {
		t.Fatal("leaf must be bound to the one hostname")
	}
	// The hook carries the authority, is executable, and is a script the
	// linuxserver image will run.
	hook, err := os.Stat(filepath.Join(dir, filepath.FromSlash(Hook)))
	if err != nil {
		t.Fatal(err)
	}
	if hook.Mode().Perm()&0o111 == 0 {
		t.Errorf("hook should be executable, got %v", hook.Mode().Perm())
	}
	script, _ := os.ReadFile(filepath.Join(dir, filepath.FromSlash(Hook)))
	if !strings.HasPrefix(string(script), "#!/bin/bash\n") {
		t.Error("hook should start with a bash shebang")
	}
	if !strings.Contains(string(script), string(caPEM)) {
		t.Error("hook should embed ca.crt")
	}
	if !strings.Contains(string(script), "update-ca-certificates") {
		t.Error("hook should run update-ca-certificates")
	}
	if err := Generate(dir, "metadata-static.plex.tv"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second run must refuse to overwrite, got %v", err)
	}
}
