package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/santiagosayshey/understudy/internal/certs"
	"github.com/santiagosayshey/understudy/internal/state"
)

const host = "metadata-static.plex.tv"

func writeImage(t *testing.T, file string, w, h int, asPNG bool) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	var buf bytes.Buffer
	if asPNG {
		png.Encode(&buf, img)
	} else {
		jpeg.Encode(&buf, img, nil)
	}
	os.WriteFile(file, buf.Bytes(), 0o644)
	return buf.Bytes()
}

func writeState(t *testing.T, dir string, m map[string]string) {
	t.Helper()
	f := &state.File{Version: 1, Ran: "2026-09-08T00:00:00Z"}
	for path, img := range m {
		f.Entries = append(f.Entries, state.Entry{Name: img, Image: img, Path: path})
	}
	b, _ := json.Marshal(f)
	os.MkdirAll(dir, 0o755)
	// write, then bump the mtime past the previous write so a fast test still
	// registers a change
	os.WriteFile(filepath.Join(dir, state.FileName), b, 0o644)
	future := time.Now().Add(time.Second)
	os.Chtimes(filepath.Join(dir, state.FileName), future, future)
}

func TestProxy(t *testing.T) {
	root := t.TempDir()
	certDir, stateDir, portraits := filepath.Join(root, "certs"), filepath.Join(root, "state"), filepath.Join(root, "portraits")
	if err := certs.Generate(certDir, host); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(portraits, 0o755)
	square := writeImage(t, filepath.Join(portraits, "square.png"), 300, 300, true)
	writeImage(t, filepath.Join(portraits, "tall.jpg"), 400, 600, false)
	writeState(t, stateDir, map[string]string{
		"/f/people/aaa.jpg":     "square.png",
		"/f/people/bbb.jpg":     "tall.jpg",
		"/f/people/missing.jpg": "nope.jpg",
	})

	// the fake CDN, with its own certificate the proxy is told to trust
	cdn := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/9/people/real.jpg" {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("X-Origin", "cdn")
			io.WriteString(w, "CDN BYTES")
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer cdn.Close()
	roots := x509.NewCertPool()
	roots.AddCert(cdn.Certificate())

	p, err := New(Options{
		CertFile: filepath.Join(certDir, "leaf.crt"), KeyFile: filepath.Join(certDir, "leaf.key"),
		CDN: cdn.URL, StateDir: stateDir, Portraits: portraits, Poll: 30 * time.Millisecond,
		UpstreamRoots: roots, Logger: log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Serve(ctx, l)

	// a client that trusts our authority and thinks it is talking to the CDN
	caPEM, _ := os.ReadFile(filepath.Join(certDir, "ca.crt"))
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig:   &tls.Config{RootCAs: pool, ServerName: host},
		ForceAttemptHTTP2: true,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, l.Addr().String())
		},
	}}
	get := func(method, path string) (*http.Response, []byte) {
		t.Helper()
		req, _ := http.NewRequest(method, "https://"+host+path, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res, b
	}

	res, body := get("GET", "/f/people/aaa.jpg")
	if res.StatusCode != 200 || !bytes.Equal(body, square) || res.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("square hit: %d %s %d bytes", res.StatusCode, res.Header.Get("Content-Type"), len(body))
	}
	if res.ProtoMajor != 2 {
		t.Errorf("Plex speaks HTTP/2; got %s", res.Proto)
	}

	res, body = get("GET", "/f/people/bbb.jpg")
	img, format, err := image.Decode(bytes.NewReader(body))
	if err != nil || res.StatusCode != 200 || format != "jpeg" || img.Bounds().Dx() != 400 || img.Bounds().Dy() != 400 {
		t.Fatalf("tall hit should be a 400x400 jpeg: %d %s %v", res.StatusCode, format, err)
	}
	if res.Header.Get("ETag") == "" {
		t.Error("cropped image should carry an ETag")
	}

	res, body = get("GET", "/9/people/real.jpg")
	if res.StatusCode != 200 || string(body) != "CDN BYTES" || res.Header.Get("X-Origin") != "cdn" {
		t.Fatalf("miss should pass through untouched: %d %q", res.StatusCode, body)
	}
	if res, _ := get("GET", "/9/people/unknown.jpg"); res.StatusCode != 404 {
		t.Fatalf("CDN 404 should pass through: %d", res.StatusCode)
	}
	if res, _ := get("GET", "/f/people/missing.jpg"); res.StatusCode != 404 {
		t.Fatalf("missing image file: %d", res.StatusCode)
	}
	if res, _ := get("POST", "/f/people/aaa.jpg"); res.StatusCode != 405 {
		t.Fatalf("POST: %d", res.StatusCode)
	}
	if res, body := get("HEAD", "/f/people/aaa.jpg"); res.StatusCode != 200 || len(body) != 0 {
		t.Fatalf("HEAD: %d %d bytes", res.StatusCode, len(body))
	}

	// the resolving job rewrites the state: aaa moves, bbb goes
	writeState(t, stateDir, map[string]string{"/f/people/ccc.jpg": "square.png"})
	deadline := time.Now().Add(3 * time.Second)
	for {
		res, _ := get("GET", "/f/people/ccc.jpg")
		if res.StatusCode == 200 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("state change not picked up")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if res, _ := get("GET", "/f/people/aaa.jpg"); res.StatusCode != 404 {
		t.Fatalf("old path should now miss and reach the CDN: %d", res.StatusCode)
	}
}
