package api

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"gopkg.in/yaml.v3"

	"github.com/santiagosayshey/understudy/internal/config"
	"github.com/santiagosayshey/understudy/internal/crop"
)

// Change is one staged edit to the configuration: a new portrait for a
// person, or the removal of their override. Changes are held here, in
// memory, until the user applies them; nothing touches the configuration
// before that.
type Change struct {
	Key      string    `json:"key"`
	Name     string    `json:"name"`
	TagKey   string    `json:"tagKey,omitempty"`
	Kind     string    `json:"kind"` // set or remove
	Image    string    `json:"image,omitempty"`
	Path     string    `json:"path,omitempty"`
	StagedAt time.Time `json:"stagedAt"`
	portrait []byte
}

// Upload is a file the browser sent, kept until it is cropped or forgotten.
type Upload struct {
	crop.Info
	data []byte
	at   time.Time
}

// Staging holds uploads and pending changes for one editor process.
type Staging struct {
	mu      sync.Mutex
	uploads map[string]*Upload
	changes map[string]*Change // by actor key
}

func NewStaging() *Staging {
	return &Staging{uploads: map[string]*Upload{}, changes: map[string]*Change{}}
}

func (s *Staging) AddUpload(id string, data []byte, info crop.Info) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, u := range s.uploads { // forget uploads older than an hour
		if time.Since(u.at) > time.Hour {
			delete(s.uploads, k)
		}
	}
	s.uploads[id] = &Upload{Info: info, data: data, at: time.Now()}
}

func (s *Staging) Upload(id string) (*Upload, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.uploads[id]
	return u, ok
}

// Stage records a change for the actor, replacing any earlier one.
func (s *Staging) Stage(c *Change) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.StagedAt = time.Now()
	s.changes[c.Key] = c
}

func (s *Staging) Discard(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.changes, key)
}

func (s *Staging) Change(key string) (*Change, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.changes[key]
	return c, ok
}

// Changes lists what is staged, oldest first.
func (s *Staging) Changes() []*Change {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Change, 0, len(s.changes))
	for _, c := range s.changes {
		out = append(out, c)
	}
	sortChanges(out)
	return out
}

func sortChanges(cs []*Change) {
	for i := 1; i < len(cs); i++ {
		for j := i; j > 0 && cs[j].StagedAt.Before(cs[j-1].StagedAt); j-- {
			cs[j], cs[j-1] = cs[j-1], cs[j]
		}
	}
}

// Applied is what one apply did.
type Applied struct {
	Written []string `json:"written"`
	Removed []string `json:"removed"`
}

// Apply writes every staged change to the configuration file and the
// portraits directory, then forgets them. The file is rewritten from its
// parsed form, so a hand-written comment does not survive; the entries and
// their order do.
func (s *Staging) Apply(ctx context.Context, configFile, portraits string) (Applied, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := config.Load(configFile)
	if err != nil {
		return Applied{}, err
	}
	var done Applied
	pending := make([]*Change, 0, len(s.changes))
	for _, c := range s.changes {
		pending = append(pending, c)
	}
	sortChanges(pending)
	for _, c := range pending {
		idx := indexOf(cfg.People, c.Name, c.TagKey)
		switch c.Kind {
		case "set":
			file := filepath.Join(portraits, filepath.FromSlash(c.Image))
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				return done, err
			}
			if err := os.WriteFile(file, c.portrait, 0o644); err != nil {
				return done, err
			}
			entry := config.Entry{Name: c.Name, TagKey: c.TagKey, Image: c.Image}
			if idx >= 0 {
				old := cfg.People[idx].Image
				cfg.People[idx] = entry
				if old != c.Image {
					os.Remove(filepath.Join(portraits, filepath.FromSlash(old)))
				}
			} else {
				cfg.People = append(cfg.People, entry)
			}
			done.Written = append(done.Written, c.Name)
		case "remove":
			if idx >= 0 {
				os.Remove(filepath.Join(portraits, filepath.FromSlash(cfg.People[idx].Image)))
				cfg.People = append(cfg.People[:idx], cfg.People[idx+1:]...)
				done.Removed = append(done.Removed, c.Name)
			}
		}
	}
	if err := writeConfig(configFile, cfg); err != nil {
		return done, err
	}
	s.changes = map[string]*Change{}
	return done, nil
}

func writeConfig(file string, cfg *config.Config) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	enc.Close()
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// indexOf finds the entry for a person. Two people can share a name, so a
// known person id wins: an entry with a different id is a different person,
// and only an entry with no id at all is matched by name.
func indexOf(entries []config.Entry, name, tagKey string) int {
	for i, e := range entries {
		if tagKey != "" && e.TagKey != "" {
			if e.TagKey == tagKey {
				return i
			}
			continue
		}
		if strings.EqualFold(strings.TrimSpace(e.Name), name) {
			return i
		}
	}
	return -1
}

var unsafe = regexp.MustCompile(`[^a-z0-9]+`)

// Slug names a portrait file from a person's name: "Cailee Spaeny" becomes
// cailee-spaeny.jpg and "Raúl Castillo" raul-castillo.jpg. When the name is
// shared, pass the person id to keep the files apart. Its tail is used, since
// the head of a Plex id is a timestamp that many ids share.
func Slug(name, tagKey string) string {
	s := slug(name)
	if tagKey != "" {
		s += "-" + tagKey[max(0, len(tagKey)-8):]
	}
	return s + ".jpg"
}

func slug(name string) string {
	plain, _, err := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), name)
	if err != nil {
		plain = name
	}
	s := strings.Trim(unsafe.ReplaceAllString(strings.ToLower(plain), "-"), "-")
	if s == "" {
		s = "portrait"
	}
	return s
}
