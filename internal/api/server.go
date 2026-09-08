package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/santiagosayshey/understudy/internal/config"
	"github.com/santiagosayshey/understudy/internal/state"
)

// Server is the editor's API. Config and state are read fresh on each
// request that needs them, because sync and the user's editor may change
// them while the process runs.
type Server struct {
	Version   string
	Listing   *Listing
	Images    *Images
	Config    string // path to configuration.yml
	Portraits string
	StateDir  string
}

// Handler routes /api.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", s.status)
	mux.HandleFunc("GET /api/actors", s.actors)
	mux.HandleFunc("GET /api/actors/{key}", s.actor)
	mux.HandleFunc("POST /api/actors/refresh", s.refresh)
	mux.HandleFunc("GET /api/images/cdn", s.cdnImage)
	mux.HandleFunc("GET /api/images/poster/{ratingKey}", s.poster)
	mux.HandleFunc("GET /api/images/portrait", s.portrait)
	return mux
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	entries, _ := s.entries()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":   s.Version,
		"listing":   s.Listing.Status(),
		"overrides": len(entries),
	})
}

// actors searches the listing and marks people who already have an override.
func (s *Server) actors(w http.ResponseWriter, r *http.Request) {
	st := s.Listing.Status()
	if !st.Loaded {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "the actor listing is still loading", "listing": st})
		return
	}
	entries, _ := s.entries()
	stateFile, _ := state.Load(s.StateDir)
	type result struct {
		Actor
		Override bool `json:"override"`
		Drift    bool `json:"drift"`
	}
	matches, total := s.Listing.Search(r.URL.Query().Get("q"), 60)
	var out []result
	for _, a := range matches {
		res := result{Actor: a}
		if e := entryFor(entries, a.Name); e != nil {
			res.Override = true
			if se := stateEntryFor(stateFile, e); se != nil && se.Path != "" && se.Path != a.Path {
				res.Drift = true
			}
		}
		out = append(out, res)
	}
	if out == nil {
		out = []result{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": out, "total": total})
}

// actor is one person with their titles, the configuration entry if any,
// and whether the state's path has drifted from the live one.
func (s *Server) actor(w http.ResponseWriter, r *http.Request) {
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
	entries, _ := s.entries()
	stateFile, _ := state.Load(s.StateDir)
	out := map[string]any{"actor": d}
	if e := entryFor(entries, d.Name); e != nil {
		override := map[string]any{"name": e.Name, "tagKey": e.TagKey, "image": e.Image}
		if se := stateEntryFor(stateFile, e); se != nil {
			override["path"] = se.Path
			override["resolved"] = se.Resolved
			override["history"] = se.History
			override["problem"] = se.Problem
			override["drift"] = se.Path != "" && se.Path != d.Path
		}
		out["override"] = override
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) poster(w http.ResponseWriter, r *http.Request) {
	width, _ := strconv.Atoi(r.URL.Query().Get("w"))
	if width <= 0 || width > 800 {
		width = 200
	}
	b, err := s.Images.Poster(r.Context(), r.PathValue("ratingKey"), width)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(b)
}

// portrait serves an image from the portraits directory, for showing the
// current override.
func (s *Server) portrait(w http.ResponseWriter, r *http.Request) {
	file, ok := config.ImagePath(s.Portraits, r.URL.Query().Get("image"))
	if !ok {
		http.Error(w, "bad image path", http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, file)
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	go s.Listing.Refresh(r.Context())
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) cdnImage(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	width, _ := strconv.Atoi(r.URL.Query().Get("w"))
	if width <= 0 || width > 1200 {
		width = 400
	}
	b, err := s.Images.CDN(r.Context(), path, width)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(b)
}

func (s *Server) entries() ([]config.Entry, error) {
	cfg, err := config.Load(s.Config)
	if err != nil {
		return nil, err
	}
	return cfg.People, nil
}

func entryFor(entries []config.Entry, name string) *config.Entry {
	for i := range entries {
		if strings.EqualFold(strings.TrimSpace(entries[i].Name), name) {
			return &entries[i]
		}
	}
	return nil
}

func stateEntryFor(f *state.File, e *config.Entry) *state.Entry {
	if f == nil {
		return nil
	}
	for i := range f.Entries {
		se := &f.Entries[i]
		if e.TagKey != "" && se.TagKey == e.TagKey || strings.EqualFold(se.Name, e.Name) {
			return se
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
