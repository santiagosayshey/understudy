package plex_test

import (
	"context"
	"testing"

	"github.com/santiagosayshey/understudy/internal/plex"
	"github.com/santiagosayshey/understudy/internal/plex/plextest"
)

func TestClient(t *testing.T) {
	srv := plextest.NewServer(t, plextest.Sample())
	c := plex.New(srv.URL, "")
	ctx := context.Background()

	sections, err := c.Sections(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 2 {
		t.Fatalf("want the movie and show sections only, got %v", sections)
	}

	actors, err := c.Actors(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	var cailee, noPhoto *plex.Actor
	for i := range actors {
		switch actors[i].Name {
		case "Cailee Spaeny":
			cailee = &actors[i]
		case "Michael Burnell":
			noPhoto = &actors[i]
		}
	}
	if cailee == nil || cailee.Key != "27126" {
		t.Fatalf("Cailee Spaeny missing or wrong key: %+v", cailee)
	}
	if p, ok := plex.Path(cailee.Thumb); !ok || p != "/f/people/fe158d30be9278d335acd7a92037b20b.jpg" {
		t.Fatalf("path: %q %v", p, ok)
	}
	if noPhoto == nil || noPhoto.Thumb != "" {
		t.Fatalf("placeholder icon should read as no thumb: %+v", noPhoto)
	}

	titles, err := c.Titles(ctx, "1", "27126")
	if err != nil {
		t.Fatal(err)
	}
	if len(titles) != 2 || titles[0].Name != "Priscilla" || titles[0].Year != 2023 {
		t.Fatalf("titles: %+v", titles)
	}

	roles, err := c.Roles(ctx, "19335")
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range roles {
		if r.Key == "27126" && r.TagKey == "5d7769e1fb0d55001f533216" {
			found = true
		}
	}
	if !found {
		t.Fatalf("numeric role id not decoded: %+v", roles)
	}
}
