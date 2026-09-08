// Package resolve turns configuration entries into people Plex knows: which
// actor a name means, Plex's id for them, and the CDN path Plex uses for their
// portrait right now. Everything it cannot decide becomes a reported problem,
// never a guess.
package resolve

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/santiagosayshey/understudy/internal/config"
	"github.com/santiagosayshey/understudy/internal/plex"
)

// Person is a resolved entry.
type Person struct {
	Name   string   // as Plex spells it
	Key    string   // server-local tag id
	TagKey string   // Plex's person id
	Path   string   // CDN path of the current portrait
	Titles []string // a few titles they appear in, for the report
}

// Kind classifies a problem so callers can act on it without parsing text.
type Kind string

const (
	Unknown   Kind = "unknown"   // no actor with that name
	Ambiguous Kind = "ambiguous" // several actors with that name and no tagKey to choose
	Mismatch  Kind = "mismatch"  // the name exists but not with the given tagKey
	NoPhoto   Kind = "nophoto"   // Plex has no portrait for the person, so nothing is requested
	Duplicate Kind = "duplicate" // another entry resolved to the same person
)

// Problem explains why an entry did not resolve.
type Problem struct {
	Kind   Kind
	Detail string
}

// Outcome is the result for one entry: a Person, or a Problem.
type Outcome struct {
	Entry   config.Entry
	Person  *Person
	Problem *Problem
}

// Resolver resolves against one Plex server.
type Resolver struct {
	Plex *plex.Client
}

type candidate struct {
	plex.Actor
	sections []plex.Section
	tagKey   string
	titles   []string
	loaded   bool
}

// Resolve loads the actor listings once and resolves every entry against
// them. The listings are the only place a current portrait path exists.
func (r *Resolver) Resolve(ctx context.Context, entries []config.Entry) ([]Outcome, error) {
	sections, err := r.Plex.Sections(ctx)
	if err != nil {
		return nil, err
	}
	byKey := map[string]*candidate{}
	byName := map[string][]*candidate{}
	for _, s := range sections {
		actors, err := r.Plex.Actors(ctx, s.Key)
		if err != nil {
			return nil, err
		}
		for _, a := range actors {
			c, ok := byKey[a.Key]
			if !ok {
				c = &candidate{Actor: a}
				byKey[a.Key] = c
				n := norm(a.Name)
				byName[n] = append(byName[n], c)
			}
			c.sections = append(c.sections, s)
		}
	}
	names := make([]string, 0, len(byName))
	for n := range byName {
		names = append(names, n)
	}

	outcomes := make([]Outcome, len(entries))
	owner := map[string]int{} // person key -> first entry index
	for i, e := range entries {
		o := Outcome{Entry: e}
		outcomes[i] = o
		cands := byName[norm(e.Name)]
		var chosen *candidate
		switch {
		case len(cands) == 0:
			outcomes[i].Problem = &Problem{Unknown, unknownDetail(e.Name, names, byName)}
			continue
		case e.TagKey != "":
			for _, c := range cands {
				if err := r.load(ctx, c); err != nil {
					return nil, err
				}
				if c.tagKey == e.TagKey {
					chosen = c
				}
			}
			if chosen == nil {
				outcomes[i].Problem = &Problem{Mismatch, fmt.Sprintf("no %s with tagKey %s; %s", e.Name, e.TagKey, describe(cands))}
				continue
			}
		case len(cands) == 1:
			chosen = cands[0]
			if err := r.load(ctx, chosen); err != nil {
				return nil, err
			}
		default:
			for _, c := range cands {
				if err := r.load(ctx, c); err != nil {
					return nil, err
				}
			}
			outcomes[i].Problem = &Problem{Ambiguous, fmt.Sprintf("%d people share this name; add tagKey: %s", len(cands), describe(cands))}
			continue
		}
		if j, dup := owner[chosen.Key]; dup {
			outcomes[i].Problem = &Problem{Duplicate, fmt.Sprintf("same person as entry %d", j+1)}
			continue
		}
		owner[chosen.Key] = i
		p := &Person{Name: chosen.Name, Key: chosen.Key, TagKey: chosen.tagKey, Titles: chosen.titles}
		if path, ok := plex.Path(chosen.Thumb); ok {
			p.Path = path
		} else {
			outcomes[i].Problem = &Problem{NoPhoto, "Plex has no portrait for this person, so nothing is ever requested and there is nothing to override"}
		}
		outcomes[i].Person = p
	}
	return outcomes, nil
}

// load fetches what the listing lacks: the person id, which only appears on
// a title's cast, and a few titles for the report.
func (r *Resolver) load(ctx context.Context, c *candidate) error {
	if c.loaded {
		return nil
	}
	c.loaded = true
	for _, s := range c.sections {
		titles, err := r.Plex.Titles(ctx, s.Key, c.Key)
		if err != nil {
			return err
		}
		for _, t := range titles {
			if c.tagKey == "" {
				roles, err := r.Plex.Roles(ctx, t.RatingKey)
				if err != nil {
					return err
				}
				for _, role := range roles {
					if role.Key == c.Key {
						c.tagKey = role.TagKey
					}
				}
			}
			if len(c.titles) < 3 {
				c.titles = append(c.titles, fmt.Sprintf("%s (%d)", t.Name, t.Year))
			}
		}
	}
	return nil
}

// describe lists candidates in a stable order so reports and tests do not
// depend on which section was listed first.
func describe(cands []*candidate) string {
	sorted := append([]*candidate(nil), cands...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].tagKey < sorted[j].tagKey })
	parts := make([]string, 0, len(sorted))
	for _, c := range sorted {
		parts = append(parts, fmt.Sprintf("%s (%s)", c.tagKey, strings.Join(c.titles, ", ")))
	}
	return strings.Join(parts, "; ")
}

func unknownDetail(name string, names []string, byName map[string][]*candidate) string {
	type near struct {
		name string
		d    int
	}
	var nears []near
	n := norm(name)
	for _, other := range names {
		if d := levenshtein(n, other); d <= 3 {
			nears = append(nears, near{byName[other][0].Name, d})
		}
	}
	if len(nears) == 0 {
		return "no actor with that name"
	}
	sort.Slice(nears, func(i, j int) bool {
		return nears[i].d < nears[j].d || nears[i].d == nears[j].d && nears[i].name < nears[j].name
	})
	if len(nears) > 3 {
		nears = nears[:3]
	}
	parts := make([]string, len(nears))
	for i, x := range nears {
		parts[i] = x.name
	}
	return "no actor with that name; nearest: " + strings.Join(parts, ", ")
}

func norm(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}
