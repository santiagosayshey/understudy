package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/santiagosayshey/understudy/internal/tmdb"
)

// tmdbMatch is what the actor page shows: the person TMDb most likely
// means, with their images, and everyone else of that name to switch to.
type tmdbMatch struct {
	Person     *tmdb.Detail  `json:"person"`
	Candidates []tmdb.Person `json:"candidates"`
}

// rankPeople orders search results for one of our actors: people TMDb
// knows for a title in the actor's Plex libraries first, then by TMDb's
// popularity. A shared name is rare, and the titles settle it.
func rankPeople(people []tmdb.Person, titles []Title) []tmdb.Person {
	have := map[string]bool{}
	for _, t := range titles {
		have[strings.ToLower(strings.TrimSpace(t.Name))] = true
	}
	hits := func(p tmdb.Person) int {
		n := 0
		for _, k := range p.KnownFor {
			if have[strings.ToLower(strings.TrimSpace(k))] {
				n++
			}
		}
		return n
	}
	out := append([]tmdb.Person(nil), people...)
	sort.SliceStable(out, func(i, j int) bool {
		hi, hj := hits(out[i]), hits(out[j])
		if hi != hj {
			return hi > hj
		}
		return out[i].Popularity > out[j].Popularity
	})
	return out
}

func (s *Server) noTMDb(w http.ResponseWriter) bool {
	if s.TMDb != nil {
		return false
	}
	writeJSON(w, http.StatusNotFound, map[string]any{"error": "no TMDb key"})
	return true
}

// actorTMDb finds one of our actors on TMDb by name and returns the best
// match with their images, and the other people of that name.
func (s *Server) actorTMDb(w http.ResponseWriter, r *http.Request) {
	if s.noTMDb(w) {
		return
	}
	if !s.Listing.Status().Loaded {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "the actor listing is still loading"})
		return
	}
	d, ok, err := s.Listing.Detail(r.Context(), r.PathValue("key"))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no actor with that key"})
		return
	}
	people, err := s.TMDb.Search(r.Context(), d.Name)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	match := tmdbMatch{Candidates: rankPeople(people, d.Titles)}
	if len(match.Candidates) > 0 {
		p, err := s.TMDb.Person(r.Context(), match.Candidates[0].ID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		match.Person = &p
	}
	writeJSON(w, http.StatusOK, match)
}

// tmdbPerson is one person by TMDb id, for switching to another candidate.
func (s *Server) tmdbPerson(w http.ResponseWriter, r *http.Request) {
	if s.noTMDb(w) {
		return
	}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "not a TMDb id"})
		return
	}
	p, err := s.TMDb.Person(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// tmdbPath checks a file path from TMDb: it is a bare file name under a
// leading slash, so anything else is not one of theirs.
func tmdbPath(p string) bool {
	return strings.HasPrefix(p, "/") && !strings.Contains(p[1:], "/") && !strings.Contains(p, "..")
}

// tmdbImage is a profile image downsized for the page.
func (s *Server) tmdbImage(w http.ResponseWriter, r *http.Request) {
	if s.noTMDb(w) {
		return
	}
	path := r.URL.Query().Get("path")
	width, _ := strconv.Atoi(r.URL.Query().Get("w"))
	if !tmdbPath(path) || width <= 0 || width > 1200 {
		http.Error(w, "bad path or width", http.StatusBadRequest)
		return
	}
	b, err := s.Images.TMDb(r.Context(), path, width)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(b)
}

// tmdbUpload fetches a profile image at full size and holds it as an
// upload, so choosing it is the same as choosing a file from then on.
func (s *Server) tmdbUpload(w http.ResponseWriter, r *http.Request) {
	if s.noTMDb(w) {
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !tmdbPath(body.Path) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "a TMDb file path is required"})
		return
	}
	data, err := s.TMDb.Image(r.Context(), body.Path, "original")
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	resp, err := s.addUpload(data)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "tmdb: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// TMDb returns a profile image no wider than width. The h632 size is the
// largest TMDb resizes to, and plenty for the page.
func (im *Images) TMDb(ctx context.Context, path string, width int) ([]byte, error) {
	return im.cached("tmdb:"+path+strconv.Itoa(width), func() ([]byte, error) {
		raw, err := im.tmdb.Image(ctx, path, "h632")
		if err != nil {
			return nil, err
		}
		return shrink(bytes.NewReader(raw), width)
	})
}
