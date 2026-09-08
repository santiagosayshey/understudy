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
	q = normName(q)
	if q == "" {
		return nil, 0
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	var starts, words, inside []Actor
	for _, a := range l.actors {
		n := normName(a.Name)
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

func normName(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

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
// from the first title's cast, since the listing does not carry it. A key
// the listing does not have, someone never billed in the top three, is
// built from their titles' cast lists instead; no titles means no such
// person.
func (l *Listing) Detail(ctx context.Context, key string) (Detail, bool, error) {
	a, listed := l.Get(key)
	l.mu.RLock()
	sections := append([]plex.Section(nil), l.sections...)
	l.mu.RUnlock()
	d := Detail{Actor: a, Titles: []Title{}}
	if !listed {
		d.Actor = Actor{Key: key, Libraries: []string{}}
	}
	seen := map[string]bool{}
	for _, s := range sections {
		titles, err := l.plex.Titles(ctx, s.Key, key)
		if err != nil {
			return d, true, err
		}
		if len(titles) > 0 && !listed {
			d.Libraries = append(d.Libraries, s.Title)
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
						if !listed {
							d.Name = r.Name
							if path, ok := plex.Path(r.Thumb); ok {
								d.Path = path
							}
						}
					}
				}
			}
		}
	}
	if !listed && d.Name == "" {
		return Detail{}, false, nil
	}
	sort.SliceStable(d.Titles, func(i, j int) bool { return d.Titles[i].Year > d.Titles[j].Year })
	return d, true, nil
}

// People asks Plex's search for names the listing does not have. Hits the
// listing already has are left out.
func (l *Listing) People(ctx context.Context, q string) ([]Actor, error) {
	found, err := l.plex.People(ctx, q)
	if err != nil {
		return nil, err
	}
	var out []Actor
	for _, p := range found {
		if _, ok := l.Get(p.Key); ok {
			continue
		}
		a := Actor{Key: p.Key, Name: p.Name, Libraries: []string{}}
		if path, ok := plex.Path(p.Thumb); ok {
			a.Path = path
		}
		out = append(out, a)
	}
	return out, nil
}

// TitleInfo is one movie or show as the title page sees it.
type TitleInfo struct {
	RatingKey string `json:"ratingKey"`
	Name      string `json:"name"`
	Year      int    `json:"year,omitempty"`
	Type      string `json:"type"`
	Library   string `json:"library"`
}

// CastMember is one person in a title's cast.
type CastMember struct {
	Actor
	TagKey string `json:"tagKey,omitempty"`
	Role   string `json:"role,omitempty"`
}

// Title fetches one title and its cast, each cast member matched to the
// listing when Plex lists them and built from the cast entry otherwise.
// The third result is false for no such title.
func (l *Listing) Title(ctx context.Context, ratingKey string) (TitleInfo, []CastMember, bool, error) {
	it, ok, err := l.plex.Item(ctx, ratingKey)
	if err != nil || !ok {
		return TitleInfo{}, nil, ok, err
	}
	info := TitleInfo{RatingKey: it.RatingKey, Name: it.Title, Year: it.Year, Type: it.Type, Library: it.Library}
	cast := make([]CastMember, 0, len(it.Roles))
	for _, r := range it.Roles {
		m := CastMember{TagKey: r.TagKey, Role: r.Role}
		if a, ok := l.Get(r.Key); ok {
			m.Actor = a
		} else {
			m.Actor = Actor{Key: r.Key, Name: r.Name, Libraries: []string{}}
			if path, ok := plex.Path(r.Thumb); ok {
				m.Actor.Path = path
			}
		}
		cast = append(cast, m)
	}
	return info, cast, true, nil
}
