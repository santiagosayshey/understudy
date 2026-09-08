// Package state is the resolving job's output: for every configured person,
// the CDN path Plex currently uses and how it got there. It is derived and
// machine-written, kept apart from the configuration so the configuration can
// stay read-only. Delete it and the next run recreates it.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/santiagosayshey/understudy/internal/resolve"
)

// FileName is the state file's name inside the state directory.
const FileName = "state.json"

// File is the whole state file.
type File struct {
	Version      int     `json:"version"`
	Ran          string  `json:"ran"`
	CacheCleared bool    `json:"cacheCleared"`
	Entries      []Entry `json:"entries"`
}

// Entry is one configured person. Path is the current CDN path, empty until
// the person has resolved with a photo. History holds every earlier path with
// the time it stopped being current. Problem is set when the last run could
// not resolve the entry; the rest of the entry is then the last known state.
type Entry struct {
	Name     string     `json:"name"`
	TagKey   string     `json:"tagKey,omitempty"`
	Key      string     `json:"key,omitempty"`
	Image    string     `json:"image"`
	Path     string     `json:"path,omitempty"`
	Resolved string     `json:"resolved,omitempty"`
	History  []Previous `json:"history,omitempty"`
	Problem  *Problem   `json:"problem,omitempty"`
}

type Previous struct {
	Path  string `json:"path"`
	Until string `json:"until"`
}

type Problem struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// Drift is one person whose path changed between runs.
type Drift struct {
	Name string
	From string
	To   string
}

// Changes summarises what a run did to the state.
type Changes struct {
	Added   []string // people who now have a path and did not before
	Removed []string // people no longer in the configuration
	Drifted []Drift
}

// Any reports whether the map the proxy serves from is different.
func (c Changes) Any() bool { return len(c.Added)+len(c.Removed)+len(c.Drifted) > 0 }

// Load reads the state directory. A missing file is an empty state, which is
// what a fresh install has.
func Load(dir string) (*File, error) {
	b, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return &File{Version: 1}, nil
	}
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(dir, FileName), err)
	}
	if f.Version != 1 {
		return nil, fmt.Errorf("%s: version %d is not supported", filepath.Join(dir, FileName), f.Version)
	}
	return &f, nil
}

// Save writes the file whole and atomically, so a reader never sees a partial
// one.
func (f *File) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".state-*.json")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, FileName))
}

// Apply folds a run's outcomes into the previous state and returns the new
// state and what changed. A person who resolved to a new path keeps the old
// one in their history. A person who did not resolve keeps their last known
// state and carries the problem. People no longer configured are dropped.
func Apply(prev *File, outcomes []resolve.Outcome, now time.Time) (*File, Changes) {
	stamp := now.UTC().Format(time.RFC3339)
	next := &File{Version: 1, Ran: stamp}
	var ch Changes
	used := map[int]bool{}
	for _, o := range outcomes {
		old, idx := match(prev, o, used)
		if idx >= 0 {
			used[idx] = true
		}
		e := Entry{Name: o.Entry.Name, TagKey: o.Entry.TagKey, Image: o.Entry.Image}
		if old != nil {
			e.TagKey, e.Key, e.Path, e.Resolved, e.History = old.TagKey, old.Key, old.Path, old.Resolved, old.History
			if e.TagKey == "" {
				e.TagKey = o.Entry.TagKey
			}
		}
		if o.Person != nil {
			e.Name, e.Key = o.Person.Name, o.Person.Key
			if o.Person.TagKey != "" {
				e.TagKey = o.Person.TagKey
			}
			if o.Person.Path != "" {
				switch {
				case e.Path == "":
					e.Path, e.Resolved = o.Person.Path, stamp
					ch.Added = append(ch.Added, e.Name)
				case e.Path != o.Person.Path:
					ch.Drifted = append(ch.Drifted, Drift{Name: e.Name, From: e.Path, To: o.Person.Path})
					e.History = append(e.History, Previous{Path: e.Path, Until: stamp})
					e.Path, e.Resolved = o.Person.Path, stamp
				}
			}
		}
		if o.Problem != nil {
			e.Problem = &Problem{Kind: string(o.Problem.Kind), Detail: o.Problem.Detail}
		}
		next.Entries = append(next.Entries, e)
	}
	for i, old := range prev.Entries {
		if !used[i] && old.Path != "" {
			ch.Removed = append(ch.Removed, old.Name)
		}
	}
	return next, ch
}

// match finds the previous entry for an outcome: by person id when both
// sides know it, otherwise by configured name, so a person who stops
// resolving still keeps their last known state.
func match(prev *File, o resolve.Outcome, used map[int]bool) (*Entry, int) {
	tagKey := o.Entry.TagKey
	if o.Person != nil && o.Person.TagKey != "" {
		tagKey = o.Person.TagKey
	}
	if tagKey != "" {
		for i := range prev.Entries {
			if !used[i] && prev.Entries[i].TagKey == tagKey {
				return &prev.Entries[i], i
			}
		}
	}
	for i := range prev.Entries {
		if !used[i] && strings.EqualFold(prev.Entries[i].Name, o.Entry.Name) && (prev.Entries[i].TagKey == "" || o.Entry.TagKey == "" || prev.Entries[i].TagKey == o.Entry.TagKey) {
			return &prev.Entries[i], i
		}
	}
	return nil, -1
}

// Map returns what the proxy serves from: CDN path to image path, for every
// person who currently has both.
func (f *File) Map() map[string]string {
	m := make(map[string]string, len(f.Entries))
	for _, e := range f.Entries {
		if e.Path != "" && e.Image != "" {
			m[e.Path] = e.Image
		}
	}
	return m
}
