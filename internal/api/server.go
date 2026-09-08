package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/santiagosayshey/understudy/internal/config"
	"github.com/santiagosayshey/understudy/internal/crop"
	"github.com/santiagosayshey/understudy/internal/state"
)

// Server is the editor's API. Config and state are read fresh on each
// request that needs them, because sync and the user's editor may change
// them while the process runs.
type Server struct {
	Version   string
	Listing   *Listing
	Images    *Images
	Staging   *Staging
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
	mux.HandleFunc("POST /api/uploads", s.upload)
	mux.HandleFunc("GET /api/uploads/{id}", s.uploadImage)
	mux.HandleFunc("GET /api/changes", s.changes)
	mux.HandleFunc("POST /api/changes", s.stage)
	mux.HandleFunc("DELETE /api/changes/{key}", s.discard)
	mux.HandleFunc("GET /api/changes/{key}/image", s.changeImage)
	mux.HandleFunc("POST /api/apply", s.apply)
	return mux
}

// maxUpload caps an uploaded file.
const maxUpload = 40 << 20

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxUpload+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(data) > maxUpload {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"error": "the file is over 40 MB"})
		return
	}
	info, err := crop.Decode(bytes.NewReader(data))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	id := newID()
	s.Staging.AddUpload(id, data, info)
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "width": info.Width, "height": info.Height, "format": info.Format})
}

func (s *Server) uploadImage(w http.ResponseWriter, r *http.Request) {
	u, ok := s.Staging.Upload(r.PathValue("id"))
	if !ok {
		http.Error(w, "no such upload", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/"+u.Format)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(u.data)
}

func (s *Server) changes(w http.ResponseWriter, r *http.Request) {
	cs := s.Staging.Changes()
	if cs == nil {
		cs = []*Change{}
	}
	writeJSON(w, http.StatusOK, cs)
}

// stage records a change for an actor: a crop of an upload, or a removal.
func (s *Server) stage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key    string   `json:"key"`
		Kind   string   `json:"kind"`
		Upload string   `json:"upload"`
		Crop   crop.Box `json:"crop"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad request"})
		return
	}
	d, ok, err := s.Listing.Detail(r.Context(), req.Key)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no actor with that key"})
		return
	}
	c := &Change{Key: d.Key, Name: d.Name, TagKey: d.TagKey, Path: d.Path, Kind: req.Kind}
	switch req.Kind {
	case "set":
		if d.Path == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Plex has no portrait for this person, so nothing is ever requested and there is nothing to override"})
			return
		}
		u, ok := s.Staging.Upload(req.Upload)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "the upload has expired; choose the file again"})
			return
		}
		out, err := crop.Square(u.data, req.Crop)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		c.Image = Slug(d.Name)
		c.portrait = out
	case "remove":
		entries, _ := s.entries()
		if entryFor(entries, d.Name) == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "there is no override to remove"})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "kind must be set or remove"})
		return
	}
	s.Staging.Stage(c)
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) discard(w http.ResponseWriter, r *http.Request) {
	s.Staging.Discard(r.PathValue("key"))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) changeImage(w http.ResponseWriter, r *http.Request) {
	c, ok := s.Staging.Change(r.PathValue("key"))
	if !ok || c.portrait == nil {
		http.Error(w, "no staged portrait", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(c.portrait)
}

func (s *Server) apply(w http.ResponseWriter, r *http.Request) {
	done, err := s.Staging.Apply(r.Context(), s.Config, s.Portraits)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "applied": done})
		return
	}
	if done.Written == nil {
		done.Written = []string{}
	}
	if done.Removed == nil {
		done.Removed = []string{}
	}
	writeJSON(w, http.StatusOK, done)
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	entries, _ := s.entries()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":   s.Version,
		"listing":   s.Listing.Status(),
		"overrides": len(entries),
		"pending":   len(s.Staging.Changes()),
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
		Override bool   `json:"override"`
		Image    string `json:"image,omitempty"`  // the override's file, for showing it
		Staged   string `json:"staged,omitempty"` // set or remove, when a change is pending
		StagedAt string `json:"stagedAt,omitempty"`
		Drift    bool   `json:"drift"`
	}
	matches, total := s.Listing.Search(r.URL.Query().Get("q"), 60)
	var out []result
	for _, a := range matches {
		res := result{Actor: a}
		if e := entryFor(entries, a.Name); e != nil {
			res.Override = true
			res.Image = e.Image
			if se := stateEntryFor(stateFile, e); se != nil && se.Path != "" && se.Path != a.Path {
				res.Drift = true
			}
		}
		if c, ok := s.Staging.Change(a.Key); ok {
			res.Staged = c.Kind
			res.StagedAt = c.StagedAt.Format("20060102150405")
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
	if c, ok := s.Staging.Change(d.Key); ok {
		out["staged"] = c
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
