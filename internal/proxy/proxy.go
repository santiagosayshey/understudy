// Package proxy stands in for the CDN. It answers to the CDN's hostname with
// the leaf certificate, serves a person's image when the requested path is in
// the map, and forwards everything else to the real CDN unchanged. It never
// talks to Plex's API; the map comes from the state file the resolving job
// writes.
package proxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/santiagosayshey/understudy/internal/state"
)

// Options configure one proxy.
type Options struct {
	CertFile  string
	KeyFile   string
	CDN       string // the real CDN, scheme and host
	StateDir  string
	Portraits string
	Logger    *log.Logger
	// Poll is how often the state file's modification time is checked.
	Poll time.Duration
	// UpstreamRoots replaces the system roots for verifying the CDN. Tests
	// use it; production leaves it nil.
	UpstreamRoots *x509.CertPool
}

// Proxy is a running instance.
type Proxy struct {
	opts     Options
	table    atomic.Pointer[map[string]string]
	upstream *httputil.ReverseProxy
	crops    sync.Map // image path -> *crop
	log      *log.Logger
	modTime  time.Time
}

type crop struct {
	data    []byte
	modTime time.Time
	etag    string
}

// New builds a proxy and loads the map once. A missing state file is an
// empty map, so a fresh install forwards everything.
func New(opts Options) (*Proxy, error) {
	if opts.Poll == 0 {
		opts.Poll = 2 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}
	cdn, err := url.Parse(opts.CDN)
	if err != nil || cdn.Host == "" {
		return nil, fmt.Errorf("proxy: bad CDN url %q", opts.CDN)
	}
	p := &Proxy{opts: opts, log: opts.Logger}
	p.table.Store(&map[string]string{})
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: opts.UpstreamRoots}
	p.upstream = &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(cdn)
			r.Out.Host = cdn.Host
			r.Out.Header.Del("X-Forwarded-For")
		},
		Transport: transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			p.log.Printf("upstream %s: %v", r.URL.Path, err)
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
		},
	}
	if err := p.reload(); err != nil {
		return nil, err
	}
	return p, nil
}

// reload reads the state file if it changed since the last look.
func (p *Proxy) reload() error {
	st, err := os.Stat(filepath.Join(p.opts.StateDir, state.FileName))
	switch {
	case errors.Is(err, os.ErrNotExist):
		if p.modTime.IsZero() {
			return nil
		}
		p.table.Store(&map[string]string{})
		p.modTime = time.Time{}
		p.log.Printf("state file removed; forwarding everything")
		return nil
	case err != nil:
		return err
	case st.ModTime().Equal(p.modTime):
		return nil
	}
	f, err := state.Load(p.opts.StateDir)
	if err != nil {
		p.log.Printf("state not reloaded: %v", err)
		return nil
	}
	m := f.Map()
	p.table.Store(&m)
	p.modTime = st.ModTime()
	p.log.Printf("state loaded: %d people", len(m))
	return nil
}

// Serve accepts TLS connections on l until ctx ends.
func (p *Proxy) Serve(ctx context.Context, l net.Listener) error {
	srv := &http.Server{
		Handler:           p,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		t := time.NewTicker(p.opts.Poll)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := p.reload(); err != nil {
					p.log.Printf("state: %v", err)
				}
			}
		}
	}()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	err := srv.ServeTLS(l, p.opts.CertFile, p.opts.KeyFile)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// ServeHTTP answers one request: a hit from the portraits directory, a miss
// from the CDN.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &recorder{ResponseWriter: w, status: http.StatusOK}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		rec.Header().Set("Allow", "GET, HEAD")
		http.Error(rec, "method not allowed", http.StatusMethodNotAllowed)
		p.logLine(r, rec, "refused", start)
		return
	}
	table := *p.table.Load()
	img, hit := table[r.URL.Path]
	if !hit {
		p.upstream.ServeHTTP(rec, r)
		p.logLine(r, rec, "miss", start)
		return
	}
	p.serveImage(rec, r, filepath.Join(p.opts.Portraits, filepath.FromSlash(img)))
	p.logLine(r, rec, "hit", start)
}

// serveImage serves the file as it is when it is square, and a centre-cropped
// JPEG otherwise, so a hand-managed folder needs no image editor.
func (p *Proxy) serveImage(w http.ResponseWriter, r *http.Request, file string) {
	f, err := os.Open(file)
	if err != nil {
		p.log.Printf("image %s: %v", file, err)
		http.Error(w, "image missing", http.StatusNotFound)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		http.Error(w, "image missing", http.StatusNotFound)
		return
	}
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		p.log.Printf("image %s does not decode: %v", file, err)
		http.Error(w, "image unreadable", http.StatusInternalServerError)
		return
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "image unreadable", http.StatusInternalServerError)
		return
	}
	if cfg.Width == cfg.Height {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, filepath.Base(file), st.ModTime(), f)
		return
	}
	c, err := p.cropped(file, f, st.ModTime())
	if err != nil {
		p.log.Printf("image %s: %v", file, err)
		http.Error(w, "image unreadable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("ETag", c.etag)
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, "", c.modTime, bytes.NewReader(c.data))
}

func (p *Proxy) cropped(file string, f io.Reader, modTime time.Time) (*crop, error) {
	if v, ok := p.crops.Load(file); ok {
		if c := v.(*crop); c.modTime.Equal(modTime) {
			return c, nil
		}
	}
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	side := min(b.Dx(), b.Dy())
	x0 := b.Min.X + (b.Dx()-side)/2
	y0 := b.Min.Y + (b.Dy()-side)/2
	dst := image.NewRGBA(image.Rect(0, 0, side, side))
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			dst.Set(x, y, src.At(x0+x, y0+y))
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(buf.Bytes())
	c := &crop{data: buf.Bytes(), modTime: modTime, etag: fmt.Sprintf(`"%x"`, sum[:8])}
	p.crops.Store(file, c)
	return c, nil
}

func (p *Proxy) logLine(r *http.Request, rec *recorder, outcome string, start time.Time) {
	p.log.Printf("%s %s %s %d %dB %s", r.Method, r.URL.Path, outcome, rec.status, rec.bytes, time.Since(start).Round(time.Millisecond))
}

// recorder captures the status and size for the log line.
type recorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *recorder) WriteHeader(code int) { r.status = code; r.ResponseWriter.WriteHeader(code) }
func (r *recorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}
func (r *recorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
