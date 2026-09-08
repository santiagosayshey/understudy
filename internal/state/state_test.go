package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/santiagosayshey/understudy/internal/config"
	"github.com/santiagosayshey/understudy/internal/resolve"
)

func person(name, key, tagKey, path string) resolve.Outcome {
	return resolve.Outcome{Entry: config.Entry{Name: name, Image: strings.ToLower(strings.ReplaceAll(name, " ", "-")) + ".jpg"},
		Person: &resolve.Person{Name: name, Key: key, TagKey: tagKey, Path: path}}
}

func problem(name string, kind resolve.Kind) resolve.Outcome {
	return resolve.Outcome{Entry: config.Entry{Name: name, Image: "x.jpg"}, Problem: &resolve.Problem{Kind: kind, Detail: string(kind)}}
}

func TestApply(t *testing.T) {
	t0 := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	t1 := t0.Add(24 * time.Hour)

	// first run: two people resolve, one is unknown
	s1, ch := Apply(&File{Version: 1}, []resolve.Outcome{
		person("Cailee Spaeny", "27126", "5d7769e1fb0d55001f533216", "/f/people/old.jpg"),
		person("Jacob Elordi", "6755", "5d7769e1fb0d55001f533217", "/d/people/df24.jpg"),
		problem("Cailey Spaeny", resolve.Unknown),
	}, t0)
	if len(ch.Added) != 2 || ch.Any() != true || len(ch.Drifted) != 0 {
		t.Fatalf("first run changes: %+v", ch)
	}
	if s1.Entries[0].Resolved != "2026-09-01T06:00:00Z" || s1.Entries[2].Problem == nil || s1.Entries[2].Path != "" {
		t.Fatalf("first run state: %+v", s1.Entries)
	}
	if m := s1.Map(); len(m) != 2 || m["/f/people/old.jpg"] != "cailee-spaeny.jpg" {
		t.Fatalf("map: %v", m)
	}

	// second run: Cailee drifted, Jacob unchanged, the unknown entry removed,
	// and Jacob stops resolving (renamed on Plex's side) but keeps his state
	s2, ch := Apply(s1, []resolve.Outcome{
		person("Cailee Spaeny", "27126", "5d7769e1fb0d55001f533216", "/f/people/new.jpg"),
		{Entry: config.Entry{Name: "Jacob Elordi", Image: "jacob-elordi.jpg"}, Problem: &resolve.Problem{Kind: resolve.Unknown, Detail: "no actor with that name"}},
	}, t1)
	if len(ch.Drifted) != 1 || ch.Drifted[0].From != "/f/people/old.jpg" || ch.Drifted[0].To != "/f/people/new.jpg" {
		t.Fatalf("drift: %+v", ch)
	}
	if len(ch.Added) != 0 || len(ch.Removed) != 0 {
		t.Fatalf("unexpected adds or removes: %+v", ch)
	}
	c := s2.Entries[0]
	if c.Path != "/f/people/new.jpg" || c.Resolved != "2026-09-02T06:00:00Z" || len(c.History) != 1 || c.History[0].Path != "/f/people/old.jpg" || c.History[0].Until != "2026-09-02T06:00:00Z" {
		t.Fatalf("drifted entry: %+v", c)
	}
	j := s2.Entries[1]
	if j.Path != "/d/people/df24.jpg" || j.TagKey != "5d7769e1fb0d55001f533217" || j.Problem == nil || j.Problem.Kind != "unknown" {
		t.Fatalf("unresolved entry should keep its last known state and carry the problem: %+v", j)
	}
	if m := s2.Map(); m["/f/people/new.jpg"] != "cailee-spaeny.jpg" || m["/d/people/df24.jpg"] != "jacob-elordi.jpg" || len(m) != 2 {
		t.Fatalf("map after drift: %v", m)
	}

	// third run: Jacob removed from the configuration, nothing else changed
	s3, ch := Apply(s2, []resolve.Outcome{
		person("Cailee Spaeny", "27126", "5d7769e1fb0d55001f533216", "/f/people/new.jpg"),
	}, t1.Add(time.Hour))
	if len(ch.Removed) != 1 || ch.Removed[0] != "Jacob Elordi" || len(ch.Drifted) != 0 || len(ch.Added) != 0 {
		t.Fatalf("removal: %+v", ch)
	}
	if len(s3.Entries) != 1 || s3.Entries[0].Resolved != "2026-09-02T06:00:00Z" {
		t.Fatalf("unchanged entry should keep its resolved time: %+v", s3.Entries)
	}
	if _, ch := Apply(s3, []resolve.Outcome{person("Cailee Spaeny", "27126", "5d7769e1fb0d55001f533216", "/f/people/new.jpg")}, t1.Add(2*time.Hour)); ch.Any() {
		t.Fatalf("a run with nothing changed must report no changes: %+v", ch)
	}
}

func TestSaveLoad(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	f, err := Load(dir)
	if err != nil || len(f.Entries) != 0 || f.Version != 1 {
		t.Fatalf("missing state should load as empty: %+v %v", f, err)
	}
	f.Ran = "2026-09-07T12:00:00Z"
	f.Entries = []Entry{{Name: "Cailee Spaeny", Image: "c.jpg", Path: "/f/people/x.jpg"}}
	if err := f.Save(dir); err != nil {
		t.Fatal(err)
	}
	g, err := Load(dir)
	if err != nil || g.Ran != f.Ran || len(g.Entries) != 1 || g.Entries[0].Path != "/f/people/x.jpg" {
		t.Fatalf("round trip: %+v %v", g, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
	os.WriteFile(filepath.Join(dir, FileName), []byte("{"), 0o644)
	if _, err := Load(dir); err == nil {
		t.Fatal("corrupt state should be an error, not an empty state")
	}
}
