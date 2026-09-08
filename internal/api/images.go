package api

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/image/draw"

	"github.com/santiagosayshey/understudy/internal/plex"
	"github.com/santiagosayshey/understudy/internal/tmdb"
)

// Images fetches and downsizes pictures for the page: CDN portraits so the
// page shows what Plex shows now, posters from Plex, and profile images
// from TMDb when there is a key. Results are kept in memory, since the same
// faces come up again and again while searching.
type Images struct {
	plex *plex.Client
	tmdb *tmdb.Client // nil without a key
	http *http.Client
	mu   sync.Mutex
	kept map[string][]byte
}

func NewImages(c *plex.Client, t *tmdb.Client) *Images {
	return &Images{plex: c, tmdb: t, http: &http.Client{Timeout: 30 * time.Second}, kept: map[string][]byte{}}
}

// CDN returns the portrait at a CDN path, no wider than width.
func (im *Images) CDN(ctx context.Context, path string, width int) ([]byte, error) {
	return im.cached("cdn:"+path+fmt.Sprint(width), func() ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, plex.CDN+path, nil)
		if err != nil {
			return nil, err
		}
		res, err := im.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("cdn: %s", res.Status)
		}
		return shrink(res.Body, width)
	})
}

// Poster returns a title's poster from Plex, no wider than width.
func (im *Images) Poster(ctx context.Context, ratingKey string, width int) ([]byte, error) {
	return im.cached("poster:"+ratingKey+fmt.Sprint(width), func() ([]byte, error) {
		raw, err := im.plex.Thumb(ctx, ratingKey)
		if err != nil {
			return nil, err
		}
		return shrink(bytes.NewReader(raw), width)
	})
}

func (im *Images) cached(key string, load func() ([]byte, error)) ([]byte, error) {
	im.mu.Lock()
	b, ok := im.kept[key]
	im.mu.Unlock()
	if ok {
		return b, nil
	}
	b, err := load()
	if err != nil {
		return nil, err
	}
	im.mu.Lock()
	im.kept[key] = b
	im.mu.Unlock()
	return b, nil
}

// shrink scales an image down so its longer side is at most width, and
// encodes it as JPEG. Smaller images are left at their size.
func shrink(r io.Reader, width int) ([]byte, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > width || h > width {
		if w >= h {
			h, w = h*width/w, width
		} else {
			w, h = w*width/h, width
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
