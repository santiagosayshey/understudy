// Package crop cuts a square out of an uploaded image and encodes it the way
// the portraits directory expects: 1000 pixels a side, progressive JPEG,
// quality 90. The crop box comes from the editor in source pixels, so the
// browser never re-encodes anything.
package crop

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"

	"golang.org/x/image/draw"
)

// Output is the side of every portrait written by the editor.
const Output = 1000

// MinSource is the smallest crop accepted, in source pixels. Plex asks for
// up to 360 a side on hi-dpi screens.
const MinSource = 200

// Box is the square to cut, in source pixels.
type Box struct {
	X    int `json:"x"`
	Y    int `json:"y"`
	Size int `json:"size"`
}

// Info describes an upload after decoding its header.
type Info struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
}

// Decode reads an upload's dimensions without decoding the pixels.
func Decode(r io.Reader) (Info, error) {
	cfg, format, err := image.DecodeConfig(r)
	if err != nil {
		return Info{}, fmt.Errorf("not a JPEG or PNG")
	}
	if cfg.Width*cfg.Height > 80_000_000 {
		return Info{}, fmt.Errorf("image is too large (%dx%d)", cfg.Width, cfg.Height)
	}
	return Info{Width: cfg.Width, Height: cfg.Height, Format: format}, nil
}

// Square cuts the box out of the upload and returns the encoded portrait.
func Square(upload []byte, box Box) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(upload))
	if err != nil {
		return nil, fmt.Errorf("not a JPEG or PNG")
	}
	b := src.Bounds()
	if box.Size < MinSource {
		return nil, fmt.Errorf("crop is %d px in the source; it needs at least %d", box.Size, MinSource)
	}
	if box.X < 0 || box.Y < 0 || box.X+box.Size > b.Dx() || box.Y+box.Size > b.Dy() {
		return nil, fmt.Errorf("crop box falls outside the image")
	}
	dst := image.NewRGBA(image.Rect(0, 0, Output, Output))
	region := image.Rect(b.Min.X+box.X, b.Min.Y+box.Y, b.Min.X+box.X+box.Size, b.Min.Y+box.Y+box.Size)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, region, draw.Src, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
