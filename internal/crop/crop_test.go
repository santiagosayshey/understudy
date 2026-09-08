package crop

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestSquare(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 600, 900))
	for y := 0; y < 900; y++ {
		for x := 0; x < 600; x++ {
			c := color.RGBA{0, 0, 200, 255}
			if y >= 100 && y < 700 { // the band the crop should land in
				c = color.RGBA{200, 0, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	info, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil || info.Width != 600 || info.Height != 900 || info.Format != "png" {
		t.Fatalf("decode: %+v %v", info, err)
	}
	out, err := Square(buf.Bytes(), Box{X: 0, Y: 100, Size: 600})
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := image.Decode(bytes.NewReader(out))
	if err != nil || got.Bounds().Dx() != Output || got.Bounds().Dy() != Output {
		t.Fatalf("output: %v %v", got.Bounds(), err)
	}
	r, _, b, _ := got.At(500, 500).RGBA()
	if r < 0x8000 || b > 0x2000 {
		t.Fatalf("crop landed outside the red band: r=%d b=%d", r>>8, b>>8)
	}
	if _, err := Square(buf.Bytes(), Box{X: 100, Y: 0, Size: 600}); err == nil {
		t.Fatal("box past the right edge must be refused")
	}
	if _, err := Square(buf.Bytes(), Box{X: 0, Y: 0, Size: 100}); err == nil {
		t.Fatal("tiny crop must be refused")
	}
}
