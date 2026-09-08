// Package api is the editor's backend: the actor listing held in memory,
// search over it, images downsized for the page, and the staging of changes
// to the configuration. It is the only part of Understudy that both reads
// Plex and writes the configuration.
package api

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/santiagosayshey/understudy/internal/plex"
)

// Actor is one person as the page sees them.
type Actor struct {
	Key       string   `json:"key"`
	Name      string   `json:"name"`
	Path      string   `json:"path,omitempty"`
	Libraries []string `json:"libraries"`
}

// Listing is every actor Plex knows, loaded once because a section listing
// takes several seconds, and refreshed on demand.
type Listing struct {
	plex *plex.Client

	mu       sync.RWMutex
	actors   []Actor
	byKey    map[string]*Actor
	sections []plex.Section
	loadedAt time.Time
	loading  bool
	err      string
}

func NewListing(c *plex.Client) *Listing {
	return &Listing{plex: c, byKey: map[string]*Actor{}}
}

// Status is what the page shows while the listing loads.
type Status struct {
	Loaded   bool      `json:"loaded"`
	Loading  bool      `json:"loading"`
	Error    string    `json:"error,omitempty"`
	Actors   int       `json:"actors"`
	Sections []string  `json:"libraries"`
	LoadedAt time.Time `json:"loadedAt,omitempty"`
}

func (l *Listing) Status() Status {
	l.mu.RLock()
	defer l.mu.RUnlock()
	s := Status{Loaded: !l.loadedAt.IsZero(), Loading: l.loading, Error: l.err, Actors: len(l.actors), LoadedAt: l.loadedAt}
	for _, sec := range l.sections {
		s.Sections = append(s.Sections, sec.Title)
	}
	return s
}

// Refresh reloads from Plex. Concurrent calls are collapsed into one.
func (l *Listing) Refresh(ctx context.Context) {
	l.mu.Lock()
	if l.loading {
		l.mu.Unlock()
		return
	}
	l.loading, l.err = true, ""
	l.mu.Unlock()
	defer func() {
		l.mu.Lock()
		l.loading = false
		l.mu.Unlock()
	}()

	sections, err := l.plex.Sections(ctx)
	if err != nil {
		l.fail(err)
		return
	}
	byKey := map[string]*Actor{}
	var actors []*Actor
	for _, s := range sections {
		list, err := l.plex.Actors(ctx, s.Key)
		if err != nil {
			l.fail(err)
			return
		}
		for _, a := range list {
			act, ok := byKey[a.Key]
			if !ok {
				path, _ := plex.Path(a.Thumb)
				act = &Actor{Key: a.Key, Name: a.Name, Path: path}
				byKey[a.Key] = act
				actors = append(actors, act)
			}
			act.Libraries = append(act.Libraries, s.Title)
		}
	}
	sort.Slice(actors, func(i, j int) bool { return actors[i].Name < actors[j].Name })
	flat := make([]Actor, len(actors))
	for i, a := range actors {
		flat[i] = *a
		byKey[a.Key] = &flat[i]
	}
	l.mu.Lock()
	l.actors, l.byKey, l.sections, l.loadedAt = flat, byKey, sections, time.Now()
	l.mu.Unlock()
}

func (l *Listing) fail(err error) {
	l.mu.Lock()
	l.err = err.Error()
	l.mu.Unlock()
}

// Get returns one actor by key.
func (l *Listing) Get(key string) (Actor, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	a, ok := l.byKey[key]
	if !ok {
		return Actor{}, false
	}
	return *a, true
}

// Search ranks names that start with the query first, then names with a word
// that starts with it, then names that contain it. It returns the first
// limit matches and how many there were in all.
func (l *Listing) Search(q string, limit int) ([]Actor, int) {
	q = norm(q)
	if q == "" {
		return nil, 0
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	var starts, words, inside []Actor
	for _, a := range l.actors {
		n := norm(a.Name)
		switch {
		case strings.HasPrefix(n, q):
			starts = append(starts, a)
		case wordPrefix(n, q):
			words = append(words, a)
		case strings.Contains(n, q):
			inside = append(inside, a)
		}
	}
	out := append(append(starts, words...), inside...)
	total := len(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, total
}

func wordPrefix(name, q string) bool {
	for _, w := range strings.Fields(name) {
		if strings.HasPrefix(w, q) {
			return true
		}
	}
	return false
}

func norm(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// Detail is one actor with everything the page shows to confirm identity.
type Detail struct {
	Actor
	TagKey string  `json:"tagKey,omitempty"`
	Titles []Title `json:"titles"`
}

// Title is something the actor appears in.
type Title struct {
	RatingKey string `json:"ratingKey"`
	Name      string `json:"name"`
	Year      int    `json:"year,omitempty"`
	Library   string `json:"library"`
}

// Detail fetches the actor's titles across the libraries and their person id
// from the first title's cast, since the listing does not carry it.
func (l *Listing) Detail(ctx context.Context, key string) (Detail, bool, error) {
	a, ok := l.Get(key)
	if !ok {
		return Detail{}, false, nil
	}
	l.mu.RLock()
	sections := append([]plex.Section(nil), l.sections...)
	l.mu.RUnlock()
	d := Detail{Actor: a, Titles: []Title{}}
	seen := map[string]bool{}
	for _, s := range sections {
		titles, err := l.plex.Titles(ctx, s.Key, key)
		if err != nil {
			return d, true, err
		}
		for _, t := range titles {
			if seen[t.RatingKey] {
				continue
			}
			seen[t.RatingKey] = true
			d.Titles = append(d.Titles, Title{RatingKey: t.RatingKey, Name: t.Name, Year: t.Year, Library: s.Title})
			if d.TagKey == "" {
				roles, err := l.plex.Roles(ctx, t.RatingKey)
				if err != nil {
					return d, true, err
				}
				for _, r := range roles {
					if r.Key == key {
						d.TagKey = r.TagKey
					}
				}
			}
		}
	}
	sort.SliceStable(d.Titles, func(i, j int) bool { return d.Titles[i].Year > d.Titles[j].Year })
	return d, true, nil
}
