package api

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/santiagosayshey/understudy/internal/config"
	"github.com/santiagosayshey/understudy/internal/crop"
	"github.com/santiagosayshey/understudy/internal/plex"
	"github.com/santiagosayshey/understudy/internal/plex/plextest"
	"github.com/santiagosayshey/understudy/internal/tmdb"
)

func loadedListing(t *testing.T) *Listing {
	t.Helper()
	srv := plextest.NewServer(t, plextest.Sample())
	l := NewListing(plex.New(srv.URL, ""))
	l.Refresh(context.Background())
	if st := l.Status(); !st.Loaded || st.Error != "" {
		t.Fatalf("listing did not load: %+v", st)
	}
	return l
}

func TestSearchRanking(t *testing.T) {
	l := loadedListing(t)
	names := func(q string) []string {
		out, _ := l.Search(q, 10)
		var ns []string
		for _, a := range out {
			ns = append(ns, a.Name)
		}
		return ns
	}
	if got := names("ca"); len(got) == 0 || got[0] != "Cailee Spaeny" {
		t.Errorf("prefix of the first name should rank first: %v", got)
	}
	if got := names("elordi"); len(got) != 1 || got[0] != "Jacob Elordi" {
		t.Errorf("word prefix: %v", got)
	}
	if got := names("anthony"); len(got) != 2 {
		t.Errorf("two people share the name: %v", got)
	}
	if got, total := l.Search("a", 2); len(got) != 2 || total < 3 {
		t.Errorf("limit and total: %d of %d", len(got), total)
	}
	if got := names(""); got != nil {
		t.Errorf("empty query: %v", got)
	}
	a, ok := l.Get("27126")
	if !ok || a.Path != "/f/people/fe158d30be9278d335acd7a92037b20b.jpg" || len(a.Libraries) != 1 {
		t.Errorf("get: %+v %v", a, ok)
	}
	d, ok, err := l.Detail(context.Background(), "27126")
	if err != nil || !ok || d.TagKey != "5d7769e1fb0d55001f533216" || len(d.Titles) != 2 || d.Titles[0].Name != "Civil War" {
		t.Errorf("detail: %+v %v %v", d, ok, err)
	}
}

func upload(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 600, 900))
	for y := 0; y < 900; y++ {
		for x := 0; x < 600; x++ {
			img.Set(x, y, color.RGBA{200, 30, 30, 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func TestStagingApply(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "configuration.yml")
	portraits := filepath.Join(dir, "portraits")
	os.MkdirAll(portraits, 0o755)
	os.WriteFile(cfgFile, []byte("version: 1\npeople:\n  - name: Someone Else\n    image: someone-else.jpg\n"), 0o644)
	os.WriteFile(filepath.Join(portraits, "someone-else.jpg"), upload(t), 0o644)

	s := NewStaging()
	data := upload(t)
	info, _ := crop.Decode(bytes.NewReader(data))
	s.AddUpload("u1", data, info)

	out, err := crop.Square(data, crop.Box{X: 0, Y: 100, Size: 600})
	if err != nil {
		t.Fatal(err)
	}
	s.Stage(&Change{Key: "27126", Name: "Cailee Spaeny", TagKey: "5d7769e1fb0d55001f533216", Kind: "set", Image: Slug("Cailee Spaeny", ""), portrait: out})
	s.Stage(&Change{Key: "999", Name: "Someone Else", Kind: "remove"})
	if len(s.Changes()) != 2 {
		t.Fatalf("staged: %d", len(s.Changes()))
	}
	// staging again for the same person replaces the earlier change
	s.Stage(&Change{Key: "27126", Name: "Cailee Spaeny", TagKey: "5d7769e1fb0d55001f533216", Kind: "set", Image: Slug("Cailee Spaeny", ""), portrait: out})
	if len(s.Changes()) != 2 {
		t.Fatalf("restaging must replace: %d", len(s.Changes()))
	}

	done, err := s.Apply(context.Background(), cfgFile, portraits)
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Written) != 1 || done.Written[0] != "Cailee Spaeny" || len(done.Removed) != 1 {
		t.Fatalf("applied: %+v", done)
	}
	cfg, err := config.Load(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.People) != 1 || cfg.People[0].Name != "Cailee Spaeny" || cfg.People[0].TagKey != "5d7769e1fb0d55001f533216" || cfg.People[0].Image != "cailee-spaeny.jpg" {
		t.Fatalf("configuration after apply: %+v", cfg.People)
	}
	if problems := config.Validate(cfg, portraits); len(problems) != 0 {
		t.Fatalf("written configuration must validate clean: %v", problems)
	}
	if _, err := os.Stat(filepath.Join(portraits, "someone-else.jpg")); !os.IsNotExist(err) {
		t.Fatal("removed person's image should be gone")
	}
	f, _ := os.Open(filepath.Join(portraits, "cailee-spaeny.jpg"))
	written, _ := crop.Decode(f)
	f.Close()
	if written.Width != crop.Output || written.Height != crop.Output {
		t.Fatalf("written portrait: %+v", written)
	}
	if len(s.Changes()) != 0 {
		t.Fatal("changes must be forgotten after apply")
	}
	raw, _ := os.ReadFile(cfgFile)
	if !strings.HasPrefix(string(raw), "version: 1\npeople:\n") {
		t.Fatalf("file shape: %q", raw)
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"Cailee Spaeny": "cailee-spaeny.jpg", "Raúl Castillo": "raul-castillo.jpg", "K Callan": "k-callan.jpg", "  ": "portrait.jpg"} {
		if got := Slug(in, ""); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
	if got := Slug("Anthony Edwards", "5d776825880197001ec9003b"); got != "anthony-edwards-1ec9003b.jpg" {
		t.Errorf("shared name: %q", got)
	}
}

// Two people with the same name are two entries, two files, and never
// overwrite each other.
func TestApplySharedName(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "configuration.yml")
	portraits := filepath.Join(dir, "portraits")
	os.MkdirAll(portraits, 0o755)
	os.WriteFile(cfgFile, []byte("version: 1\npeople: []\n"), 0o644)
	data := upload(t)
	out, _ := crop.Square(data, crop.Box{X: 0, Y: 0, Size: 600})
	s := NewStaging()
	s.Stage(&Change{Key: "300", Name: "Anthony Edwards", TagKey: "5d776825880197001ec9003b", Kind: "set", Image: Slug("Anthony Edwards", "5d776825880197001ec9003b"), portrait: out})
	s.Stage(&Change{Key: "301", Name: "Anthony Edwards", TagKey: "5d776825880197001ec901a4", Kind: "set", Image: Slug("Anthony Edwards", "5d776825880197001ec901a4"), portrait: out})
	if _, err := s.Apply(context.Background(), cfgFile, portraits); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load(cfgFile)
	if len(cfg.People) != 2 || cfg.People[0].Image == cfg.People[1].Image {
		t.Fatalf("shared name must produce two entries with distinct files: %+v", cfg.People)
	}
	if problems := config.Validate(cfg, portraits); len(problems) != 0 {
		t.Fatalf("must validate: %v", problems)
	}
	// removing one leaves the other untouched
	s.Stage(&Change{Key: "300", Name: "Anthony Edwards", TagKey: "5d776825880197001ec9003b", Kind: "remove"})
	if _, err := s.Apply(context.Background(), cfgFile, portraits); err != nil {
		t.Fatal(err)
	}
	cfg, _ = config.Load(cfgFile)
	if len(cfg.People) != 1 || cfg.People[0].TagKey != "5d776825880197001ec901a4" {
		t.Fatalf("wrong person removed: %+v", cfg.People)
	}
}

func TestTitle(t *testing.T) {
	l := loadedListing(t)
	info, cast, ok, err := l.Title(context.Background(), "12583")
	if err != nil || !ok {
		t.Fatalf("title: ok=%v err=%v", ok, err)
	}
	if info.Name != "Civil War" || info.Year != 2024 || info.Library != "Movies" || info.Type != "movie" {
		t.Errorf("got %+v", info)
	}
	var cailee *CastMember
	for i := range cast {
		if cast[i].Name == "Cailee Spaeny" {
			cailee = &cast[i]
		}
	}
	if cailee == nil {
		t.Fatalf("Cailee Spaeny missing from the cast: %+v", cast)
	}
	if cailee.Key != "27126" || cailee.Role != "Jessie" || cailee.TagKey == "" || cailee.Path == "" {
		t.Errorf("got %+v", *cailee)
	}
	var tramell *CastMember
	for i := range cast {
		if cast[i].Name == "Tramell Tillman" {
			tramell = &cast[i]
		}
	}
	if tramell == nil || tramell.Key != "24899" || tramell.Path == "" {
		t.Errorf("cast Plex does not list should still be in the cast: %+v", tramell)
	}
	if _, _, ok, err := l.Title(context.Background(), "999"); err != nil || ok {
		t.Errorf("unknown title: ok=%v err=%v", ok, err)
	}
}

func TestUnlistedPerson(t *testing.T) {
	l := loadedListing(t)
	if _, ok := l.Get("24899"); ok {
		t.Fatal("the fake should not list Tramell Tillman")
	}
	d, ok, err := l.Detail(context.Background(), "24899")
	if err != nil || !ok {
		t.Fatalf("detail: ok=%v err=%v", ok, err)
	}
	if d.Name != "Tramell Tillman" || d.TagKey != "5d776825880197001ec9aaaa" || d.Path == "" || len(d.Titles) != 1 || len(d.Libraries) != 1 {
		t.Errorf("got %+v", d)
	}
	if _, ok, err := l.Detail(context.Background(), "999999"); err != nil || ok {
		t.Errorf("unknown key: ok=%v err=%v", ok, err)
	}
	people, err := l.People(context.Background(), "tillman")
	if err != nil || len(people) != 1 || people[0].Key != "24899" {
		t.Errorf("search: %+v %v", people, err)
	}
	if people, _ := l.People(context.Background(), "cailee"); len(people) != 0 {
		t.Errorf("listed people should not come back from search: %+v", people)
	}
}

func TestRankPeople(t *testing.T) {
	titles := []Title{{Name: "Priscilla"}, {Name: "Civil War"}}
	people := []tmdb.Person{
		{ID: 1, Name: "A", Popularity: 50, KnownFor: []string{"Something Else"}},
		{ID: 2, Name: "B", Popularity: 1, KnownFor: []string{"priscilla "}},
		{ID: 3, Name: "C", Popularity: 5, KnownFor: []string{"Civil War", "Priscilla"}},
		{ID: 4, Name: "D", Popularity: 60},
	}
	var ids []int
	for _, p := range rankPeople(people, titles) {
		ids = append(ids, p.ID)
	}
	if want := []int{3, 2, 4, 1}; !slices.Equal(ids, want) {
		t.Errorf("titles in the libraries first, case and space aside, then popularity: got %v want %v", ids, want)
	}
	if got := rankPeople(nil, titles); len(got) != 0 {
		t.Errorf("nothing in, nothing out: %v", got)
	}
}

func TestTMDbPath(t *testing.T) {
	for p, want := range map[string]bool{
		"/abc.jpg": true, "abc.jpg": false, "/a/b.jpg": false, "/../x.jpg": false, "": false, "/": true,
	} {
		if got := tmdbPath(p); got != want {
			t.Errorf("%q: got %v want %v", p, got, want)
		}
	}
}

// A profile image fetched from TMDb becomes an upload like any other, so
// the crop and the stage that follow need nothing new.
func TestTMDbUpload(t *testing.T) {
	data := upload(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/original/p.jpg" {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	c := tmdb.New("k")
	c.Images = srv.URL
	s := &Server{Staging: NewStaging(), TMDb: c}

	raw, err := c.Image(context.Background(), "/p.jpg", "original")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := s.addUpload(raw)
	if err != nil {
		t.Fatal(err)
	}
	u, ok := s.Staging.Upload(resp["id"].(string))
	if !ok || u.Width != 600 || u.Height != 900 || resp["format"] != "jpeg" {
		t.Errorf("upload: %+v %v %v", u, ok, resp)
	}
	if _, err := s.addUpload([]byte("not an image")); err == nil {
		t.Error("what TMDb returns still has to decode")
	}
}
