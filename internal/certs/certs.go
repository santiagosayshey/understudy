// Package certs makes the two certificates the proxy needs: a private
// authority, and a leaf for the CDN hostname signed by it. The authority is
// what the Plex container is told to trust; the leaf is what the proxy
// presents. Both are EC P-256 and valid for ten years, which Plex accepts.
// The authority reaches Plex as a startup script with the certificate
// embedded, so the Plex container mounts one directory and nothing else.
package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// Files are the names written into the output directory.
var Files = []string{"ca.crt", "ca.key", "leaf.crt", "leaf.key"}

// Hook is the startup script for the Plex container, under the output
// directory. Its directory is what the linuxserver image mounts at
// /custom-cont-init.d; it runs on every start, so an image upgrade trusts
// the authority again before Plex serves anything.
const Hook = "plex/10-understudy-ca.sh"

// Generate writes the four files and the hook into dir. It refuses to
// overwrite any of them, because a new authority would silently stop
// matching the one installed in the Plex container.
func Generate(dir, hostname string) error {
	for _, f := range append(append([]string{}, Files...), Hook) {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return fmt.Errorf("%s already exists; remove the old certificates first if you really want new ones", filepath.Join(dir, f))
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	now := time.Now()
	tenYears := now.AddDate(10, 0, 0)

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          serial(),
		Subject:               pkix.Name{CommonName: "Understudy CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              tenYears,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		return err
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: hostname},
		DNSNames:     []string{hostname},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     tenYears,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		return err
	}

	for name, w := range map[string]func() ([]byte, string, error){
		"ca.crt":   func() ([]byte, string, error) { return caDER, "CERTIFICATE", nil },
		"leaf.crt": func() ([]byte, string, error) { return leafDER, "CERTIFICATE", nil },
		"ca.key": func() ([]byte, string, error) {
			b, err := x509.MarshalECPrivateKey(caKey)
			return b, "EC PRIVATE KEY", err
		},
		"leaf.key": func() ([]byte, string, error) {
			b, err := x509.MarshalECPrivateKey(leafKey)
			return b, "EC PRIVATE KEY", err
		},
	} {
		der, typ, err := w()
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if typ == "EC PRIVATE KEY" {
			mode = 0o600
		}
		if err := os.WriteFile(filepath.Join(dir, name), pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), mode); err != nil {
			return err
		}
	}
	hook := filepath.Join(dir, filepath.FromSlash(Hook))
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		return err
	}
	return os.WriteFile(hook, hookScript(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})), 0o755)
}

// hookScript is the Plex container's startup script with the authority
// embedded, so the container needs no other file from Understudy.
func hookScript(caPEM []byte) []byte {
	return []byte(`#!/bin/bash
# Makes Plex trust Understudy's certificate authority. Written by understudy
# cert; mount the directory this file is in at /custom-cont-init.d in the Plex
# container (linuxserver image). It runs on every start, so an image upgrade
# is trusted again before Plex serves anything.
mkdir -p /usr/local/share/ca-certificates
cat > /usr/local/share/ca-certificates/understudy.crt <<'EOF'
` + string(caPEM) + `EOF
if update-ca-certificates >/dev/null 2>&1; then
    echo "[understudy] certificate authority installed"
else
    echo "[understudy] update-ca-certificates failed; Plex will not trust the proxy"
fi
`)
}

func serial() *big.Int {
	n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		panic(err)
	}
	return n
}
