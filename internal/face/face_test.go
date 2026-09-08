package face

import (
	"image"
	"image/color"
	"testing"
)

func TestAround(t *testing.T) {
	cases := []struct {
		name               string
		w, h, cx, cy, size int
		want               Box
	}{
		{"centred", 1000, 1000, 500, 500, 250, Box{300, 300, 400}},
		{"near the top left edge", 1000, 1000, 100, 50, 250, Box{0, 0, 400}},
		{"near the bottom right edge", 1000, 1000, 950, 980, 250, Box{600, 600, 400}},
		{"face wider than the image", 300, 600, 150, 200, 500, Box{0, 50, 300}},
	}
	for _, c := range cases {
		if got := Around(c.w, c.h, c.cx, c.cy, c.size); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestSuggestNoFace(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			img.Set(x, y, color.RGBA{uint8(x / 3), uint8(y / 2), 128, 255})
		}
	}
	if box, ok := Suggest(img); ok {
		t.Errorf("a gradient has no face, got %+v", box)
	}
}
