package resolve

import (
	"context"
	"strings"
	"testing"

	"github.com/santiagosayshey/understudy/internal/config"
	"github.com/santiagosayshey/understudy/internal/plex"
	"github.com/santiagosayshey/understudy/internal/plex/plextest"
)

func TestResolve(t *testing.T) {
	srv := plextest.NewServer(t, plextest.Sample())
	r := &Resolver{Plex: plex.New(srv.URL, "")}
	entries := []config.Entry{
		{Name: "cailee spaeny", Image: "a.jpg"},                                       // case-insensitive match
		{Name: "Cailey Spaeny", Image: "b.jpg"},                                       // misspelled
		{Name: "Anthony Edwards", Image: "c.jpg"},                                     // ambiguous
		{Name: "Anthony Edwards", Image: "d.jpg", TagKey: "5d776825880197001ec901a4"}, // disambiguated
		{Name: "Jacob Elordi", Image: "e.jpg", TagKey: "000000000000000000000000"},    // wrong tagKey
		{Name: "Michael Burnell", Image: "f.jpg"},                                     // no photo
		{Name: "Cailee Spaeny", Image: "g.jpg"},                                       // duplicate of entry 1
		{Name: "Nobody Atall", Image: "h.jpg"},                                        // nothing near
		{Name: "tramell tillman", Image: "i.jpg"},                                     // not in the listing, found by search
	}
	out, err := r.Resolve(context.Background(), entries)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		kind   Kind
		detail string
		path   string
	}{
		{"", "", "/f/people/fe158d30be9278d335acd7a92037b20b.jpg"},
		{Unknown, "nearest: Cailee Spaeny", ""},
		{Ambiguous, "5d776825880197001ec9003b (ER (1994)); 5d776825880197001ec901a4 (Top Gun (1986))", ""},
		{"", "", "/b/people/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.jpg"},
		{Mismatch, "no Jacob Elordi with tagKey 000000000000000000000000", ""},
		{NoPhoto, "no portrait", ""},
		{Duplicate, "same person as entry 1", ""},
		{Unknown, "no actor with that name", ""},
		{"", "", "/7/people/77777777777777777777777777777777.jpg"},
	}
	for i, w := range want {
		o := out[i]
		if w.kind == "" {
			if o.Problem != nil || o.Person == nil || o.Person.Path != w.path {
				t.Errorf("entry %d: want path %s, got person %+v problem %+v", i, w.path, o.Person, o.Problem)
			}
			continue
		}
		if o.Problem == nil || o.Problem.Kind != w.kind || !strings.Contains(o.Problem.Detail, w.detail) {
			t.Errorf("entry %d: want %s containing %q, got %+v", i, w.kind, w.detail, o.Problem)
		}
	}
	if out[0].Person.TagKey != "5d7769e1fb0d55001f533216" || out[0].Person.Name != "Cailee Spaeny" {
		t.Errorf("tagKey and spelling should come from Plex: %+v", out[0].Person)
	}
	if out[7].Problem.Detail != "no actor with that name" {
		t.Errorf("no near names should be reported for %q: %s", entries[7].Name, out[7].Problem.Detail)
	}
	if out[8].Person.TagKey != "5d776825880197001ec9aaaa" || out[8].Person.Name != "Tramell Tillman" {
		t.Errorf("a person found by search should carry Plex's id and spelling: %+v", out[8].Person)
	}
	if srv.Requests > 24 {
		t.Errorf("too many requests for a small library: %d", srv.Requests)
	}
}
