// Package face finds the face in an uploaded photo so the crop editor can
// open on it. The detector is pigo, a pure Go port of pico, with its
// cascade embedded in the binary: a few hundred pixel comparisons per
// window, tens of milliseconds per photo, no model to ship to the browser.
// Not finding a face is not an error; the editor falls back to its usual
// guess. The package depends on nothing of ours.
package face

import (
	_ "embed"
	"image"
	"sync"

	pigo "github.com/esimov/pigo/core"
	"golang.org/x/image/draw"
)

//go:embed facefinder
var cascade []byte

// Frame is the crop's side in faces. Measured from a set of hand-made
// crops, where the face is 60 to 80 percent of the square.
const Frame = 1.6

// minQ is the detector's confidence below which a hit is ignored. Faces in
// photographs score from the tens into the hundreds.
const minQ = 10

// detectAt is the long side the photo is scaled to before detection. The
// detector's cost grows with pixels and it needs no more than this.
const detectAt = 1000

// Box is a square in source pixels, the crop editor's unit.
type Box struct {
	X, Y, Size int
}

var classifier = sync.OnceValue(func() *pigo.Pigo {
	c, err := pigo.NewPigo().Unpack(cascade)
	if err != nil {
		panic("face: embedded cascade: " + err.Error())
	}
	return c
})

// Suggest is a square crop centred on the largest face in img, Frame faces
// wide so head and shoulders fill it, kept inside the image. The second
// result is false when no face is found.
func Suggest(img image.Image) (Box, bool) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return Box{}, false
	}
	scale := 1.0
	src := img
	if long := max(w, h); long > detectAt {
		scale = float64(long) / detectAt
		dst := image.NewRGBA(image.Rect(0, 0, int(float64(w)/scale), int(float64(h)/scale)))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
		src = dst
	}
	cols, rows := src.Bounds().Dx(), src.Bounds().Dy()
	params := pigo.CascadeParams{
		MinSize:     40,
		MaxSize:     max(cols, rows),
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{Pixels: pigo.RgbToGrayscale(src), Rows: rows, Cols: cols, Dim: cols},
	}
	c := classifier()
	var best *pigo.Detection
	for _, d := range c.ClusterDetections(c.RunCascade(params, 0), 0.2) {
		if d.Q < minQ {
			continue
		}
		if best == nil || d.Scale > best.Scale {
			d := d
			best = &d
		}
	}
	if best == nil {
		return Box{}, false
	}
	return Around(w, h, int(float64(best.Col)*scale), int(float64(best.Row)*scale), int(float64(best.Scale)*scale)), true
}

// Around is the square of Frame faces centred on a face of the given size
// at (cx, cy), moved and shrunk as needed to stay inside a w by h image.
func Around(w, h, cx, cy, size int) Box {
	side := min(int(float64(size)*Frame), w, h)
	x := min(max(cx-side/2, 0), w-side)
	y := min(max(cy-side/2, 0), h-side)
	return Box{X: x, Y: y, Size: side}
}
