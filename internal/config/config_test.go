package config

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeImage(t *testing.T, dir, name string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{200, 30, 30, 255})
		}
	}
	os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755)
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if strings.HasSuffix(name, ".png") {
		png.Encode(f, img)
	} else {
		jpeg.Encode(f, img, nil)
	}
}

func TestLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	portraits := filepath.Join(dir, "portraits")
	writeImage(t, portraits, "cailee-spaeny.jpg", 800, 800)
	writeImage(t, portraits, "tall.png", 400, 600)
	writeImage(t, portraits, "sub/elordi.jpg", 500, 500)
	os.WriteFile(filepath.Join(portraits, "text.jpg"), []byte("not an image"), 0o644)

	file := filepath.Join(dir, "configuration.yml")
	os.WriteFile(file, []byte(`version: 1
people:
  - name: Cailee Spaeny
    image: cailee-spaeny.jpg
  - name: Someone Tall
    image: tall.png
  - name: Jacob Elordi
    image: sub/elordi.jpg
    tagKey: 5d7769e1fb0d55001f533217
  - name: Missing File
    image: nope.jpg
  - name: Escapes
    image: ../etc/passwd
  - name: ""
    image: cailee-spaeny.jpg
  - name: Bad Key
    image: cailee-spaeny.jpg
    tagKey: notahexkey
  - name: cailee spaeny
    image: cailee-spaeny.jpg
  - name: Not Image
    image: text.jpg
  - name: No Image
`), 0o644)

	c, err := Load(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.People) != 10 {
		t.Fatalf("want 10 entries, got %d", len(c.People))
	}
	problems := Validate(c, portraits)
	got := map[int][]string{}
	for _, p := range problems {
		got[p.Index] = append(got[p.Index], p.Detail)
	}
	want := map[int]string{
		1: "not square",
		3: "no such file",
		4: "inside the portraits directory",
		5: "name is required",
		6: "not a Plex person id",
		7: "same person as entry 1",
		8: "does not decode",
		9: "image is required",
	}
	for i, substr := range want {
		if len(got[i]) == 0 || !strings.Contains(strings.Join(got[i], " "), substr) {
			t.Errorf("entry %d: want a problem containing %q, got %v", i, substr, got[i])
		}
	}
	for _, i := range []int{0, 2} {
		if len(got[i]) != 0 {
			t.Errorf("entry %d should be clean, got %v", i, got[i])
		}
	}
	for _, p := range problems {
		if p.Index == 1 && p.Severity != Warning {
			t.Errorf("non-square should be a warning")
		}
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"version: 2\npeople: []\n":          "not supported",
		"people: []\n":                      "not supported",
		"version: 1\npeople:\n  - nam: x\n": "field nam not found",
		"":                                  "empty",
		"version: 1\npeople: [\n":           "yaml",
	}
	for content, substr := range cases {
		f := filepath.Join(dir, "c.yml")
		os.WriteFile(f, []byte(content), 0o644)
		_, err := Load(f)
		if err == nil || !strings.Contains(err.Error(), substr) {
			t.Errorf("%q: want error containing %q, got %v", content, substr, err)
		}
	}
}
