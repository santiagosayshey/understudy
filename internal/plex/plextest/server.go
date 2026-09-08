// Package plextest is a fake Plex server for tests. It answers the four
// endpoints the client uses from an in-memory library, in the shapes the real
// server uses, including its habit of mixing string and numeric ids.
package plextest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Person is one actor in the fake library. Thumb is the full URL, or empty
// for a person with no photo. Titles are the rating keys they appear in.
type Person struct {
	Key    string
	Name   string
	TagKey string
	Thumb  string
	Titles []string
	Roles  map[string]string // character by rating key, when known
	// Unlisted people are in cast lists but not the actor listing, the way
	// the real server omits anyone never billed in the top three.
	Unlisted bool
}

// Title is one movie or show.
type Title struct {
	RatingKey string
	Name      string
	Year      int
	Section   string
}

// Library is the fake's content.
type Library struct {
	Sections []Section
	People   []Person
	Titles   []Title
}

type Section struct {
	Key, Type, Title string
}

// Server serves a Library and records how many requests it answered.
type Server struct {
	*httptest.Server
	Requests int
	lib      Library
}

func NewServer(t *testing.T, lib Library) *Server {
	t.Helper()
	s := &Server{lib: lib}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Close)
	return s
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.Requests++
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path
	switch {
	case path == "/library/sections":
		var dirs []map[string]any
		for _, sec := range s.lib.Sections {
			dirs = append(dirs, map[string]any{"key": sec.Key, "type": sec.Type, "title": sec.Title})
		}
		writeContainer(w, "Directory", dirs)
	case strings.HasSuffix(path, "/actor"):
		section := strings.TrimSuffix(strings.TrimPrefix(path, "/library/sections/"), "/actor")
		var dirs []map[string]any
		for _, p := range s.lib.People {
			if p.Unlisted || !s.inSection(p, section) {
				continue
			}
			thumb := p.Thumb
			if thumb == "" {
				thumb = "/:/resources/actor-icon.png?t=1"
			}
			dirs = append(dirs, map[string]any{"key": p.Key, "title": p.Name, "thumb": thumb})
		}
		writeContainer(w, "Directory", dirs)
	case strings.HasSuffix(path, "/all"):
		section := strings.TrimSuffix(strings.TrimPrefix(path, "/library/sections/"), "/all")
		actor := r.URL.Query().Get("actor")
		var meta []map[string]any
		for _, p := range s.lib.People {
			if p.Key != actor {
				continue
			}
			for _, rk := range p.Titles {
				if t, ok := s.title(rk); ok && t.Section == section {
					meta = append(meta, map[string]any{"ratingKey": t.RatingKey, "title": t.Name, "year": t.Year})
				}
			}
		}
		writeContainer(w, "Metadata", meta)
	case path == "/library/search":
		q := strings.ToLower(r.URL.Query().Get("query"))
		var results []map[string]any
		for _, p := range s.lib.People {
			if q == "" || !strings.Contains(strings.ToLower(p.Name), q) {
				continue
			}
			id, _ := json.Number(p.Key).Int64()
			results = append(results, map[string]any{"Directory": map[string]any{"type": "tag", "id": id, "tag": p.Name, "tagKey": p.TagKey, "thumb": p.Thumb}})
		}
		writeContainer(w, "SearchResult", results)
	case strings.HasPrefix(path, "/library/metadata/"):
		rk := strings.TrimPrefix(path, "/library/metadata/")
		title, ok := s.title(rk)
		if !ok {
			writeContainer(w, "Metadata", []map[string]any{})
			return
		}
		var roles []map[string]any
		for _, p := range s.lib.People {
			for _, t := range p.Titles {
				if t == rk {
					id, _ := json.Number(p.Key).Int64() // the real server sends role ids as numbers
					roles = append(roles, map[string]any{"id": id, "tag": p.Name, "tagKey": p.TagKey, "thumb": p.Thumb, "role": p.Roles[rk]})
				}
			}
		}
		var sectionType, sectionTitle string
		for _, sec := range s.lib.Sections {
			if sec.Key == title.Section {
				sectionType, sectionTitle = sec.Type, sec.Title
			}
		}
		writeContainer(w, "Metadata", []map[string]any{{
			"ratingKey": rk, "title": title.Name, "year": title.Year, "type": sectionType,
			"librarySectionTitle": sectionTitle, "Role": roles,
		}})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) inSection(p Person, section string) bool {
	for _, rk := range p.Titles {
		if t, ok := s.title(rk); ok && t.Section == section {
			return true
		}
	}
	return false
}

func (s *Server) title(rk string) (Title, bool) {
	for _, t := range s.lib.Titles {
		if t.RatingKey == rk {
			return t, true
		}
	}
	return Title{}, false
}

func writeContainer(w http.ResponseWriter, key string, items any) {
	json.NewEncoder(w).Encode(map[string]any{"MediaContainer": map[string]any{key: items}})
}

// Sample is a small library with the cases the resolver has to handle: a
// plain match, a shared name, a person with no photo, and a person in two
// sections.
func Sample() Library {
	return Library{
		Sections: []Section{{Key: "1", Type: "movie", Title: "Movies"}, {Key: "2", Type: "show", Title: "TV Shows"}, {Key: "5", Type: "artist", Title: "Music"}},
		Titles: []Title{
			{RatingKey: "19335", Name: "Priscilla", Year: 2023, Section: "1"},
			{RatingKey: "12583", Name: "Civil War", Year: 2024, Section: "1"},
			{RatingKey: "100", Name: "ER", Year: 1994, Section: "2"},
			{RatingKey: "101", Name: "Top Gun", Year: 1986, Section: "1"},
			{RatingKey: "102", Name: "Euphoria", Year: 2019, Section: "2"},
		},
		People: []Person{
			{Key: "27126", Name: "Cailee Spaeny", TagKey: "5d7769e1fb0d55001f533216", Thumb: "https://metadata-static.plex.tv/f/people/fe158d30be9278d335acd7a92037b20b.jpg", Titles: []string{"19335", "12583"}, Roles: map[string]string{"12583": "Jessie"}},
			{Key: "6755", Name: "Jacob Elordi", TagKey: "5d7769e1fb0d55001f533217", Thumb: "https://metadata-static.plex.tv/d/people/df243843965949c6b502cd1c0c056648.jpg", Titles: []string{"19335", "102"}},
			{Key: "300", Name: "Anthony Edwards", TagKey: "5d776825880197001ec9003b", Thumb: "https://metadata-static.plex.tv/a/people/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.jpg", Titles: []string{"100"}},
			{Key: "301", Name: "Anthony Edwards", TagKey: "5d776825880197001ec901a4", Thumb: "https://metadata-static.plex.tv/b/people/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.jpg", Titles: []string{"101"}},
			{Key: "24899", Name: "Tramell Tillman", TagKey: "5d776825880197001ec9aaaa", Thumb: "https://metadata-static.plex.tv/7/people/77777777777777777777777777777777.jpg", Titles: []string{"12583"}, Roles: map[string]string{"12583": "Bill"}, Unlisted: true},
			{Key: "400", Name: "Michael Burnell", TagKey: "5d7768cc7a53e9001e74c2d4", Thumb: "", Titles: []string{"19335"}},
		},
	}
}
