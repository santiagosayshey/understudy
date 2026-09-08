// Package config reads the configuration: configuration.yml and the portraits
// directory it names images in. It checks everything that can be checked
// without Plex. Whether a name resolves is the resolver's job.
package config

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the parsed file.
type Config struct {
	Version int     `yaml:"version"`
	People  []Entry `yaml:"people"`
}

// Entry is one person. Name is matched against Plex. TagKey is Plex's person
// id and is needed only when two people share a name. Image is a path
// relative to the portraits directory.
type Entry struct {
	Name   string `yaml:"name"`
	TagKey string `yaml:"tagKey,omitempty"`
	Image  string `yaml:"image"`
}

// Severity says whether a problem stops the entry from working.
type Severity int

const (
	Error Severity = iota
	Warning
)

// Problem is one thing wrong with one entry.
type Problem struct {
	Index    int // position in People
	Severity Severity
	Detail   string
}

func (p Problem) String() string { return p.Detail }

// Load parses the file. A missing file is an empty configuration, which is
// what a fresh install has; the editor writes the file on the first apply.
// Anything else that makes the whole file unusable is an error: unreadable,
// unparseable, or an unknown version.
func Load(file string) (*Config, error) {
	b, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Version: 1}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		if err.Error() == "EOF" {
			return nil, fmt.Errorf("%s: empty file", file)
		}
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if c.Version != 1 {
		return nil, fmt.Errorf("%s: version %d is not supported, this build reads version 1", file, c.Version)
	}
	return &c, nil
}

// Validate runs the local checks on every entry: required fields, image paths
// that stay inside the portraits directory, files that exist and decode, and
// squareness, which is a warning because a non-square file is centre-cropped
// when served. It also reports two entries that are plainly the same person.
func Validate(c *Config, portraits string) []Problem {
	var problems []Problem
	add := func(i int, s Severity, format string, args ...any) {
		problems = append(problems, Problem{Index: i, Severity: s, Detail: fmt.Sprintf(format, args...)})
	}
	seen := map[string]int{}
	for i, e := range c.People {
		if strings.TrimSpace(e.Name) == "" {
			add(i, Error, "name is required")
		}
		if e.TagKey != "" && !validTagKey(e.TagKey) {
			add(i, Error, "tagKey %q is not a Plex person id (24 hex digits)", e.TagKey)
		}
		id := strings.ToLower(strings.TrimSpace(e.Name)) + "\x00" + e.TagKey
		if j, dup := seen[id]; dup {
			add(i, Error, "same person as entry %d", j+1)
		}
		seen[id] = i
		if e.Image == "" {
			add(i, Error, "image is required")
			continue
		}
		file, ok := imagePath(portraits, e.Image)
		if !ok {
			add(i, Error, "image %q must be a relative path inside the portraits directory", e.Image)
			continue
		}
		f, err := os.Open(file)
		if err != nil {
			add(i, Error, "image %q: %v", e.Image, unwrapPath(err))
			continue
		}
		cfg, format, err := image.DecodeConfig(f)
		f.Close()
		if err != nil {
			add(i, Error, "image %q does not decode as JPEG or PNG", e.Image)
			continue
		}
		if format != "jpeg" && format != "png" {
			add(i, Error, "image %q is %s, only JPEG and PNG are served", e.Image, format)
			continue
		}
		if cfg.Width != cfg.Height {
			add(i, Warning, "image %q is %dx%d, not square; it is centre-cropped when served", e.Image, cfg.Width, cfg.Height)
		}
	}
	return problems
}

// ImagePath resolves an entry's image within the portraits directory, or
// reports that it escapes it.
func ImagePath(portraits, img string) (string, bool) { return imagePath(portraits, img) }

func imagePath(portraits, img string) (string, bool) {
	if img == "" || path.IsAbs(img) || filepath.IsAbs(img) || strings.Contains(img, "\\") {
		return "", false
	}
	clean := path.Clean("/" + img)
	if clean == "/" || strings.HasPrefix(img, "..") || strings.Contains(img, "/../") || strings.HasSuffix(img, "/..") {
		return "", false
	}
	return filepath.Join(portraits, filepath.FromSlash(clean)), true
}

func validTagKey(s string) bool {
	if len(s) != 24 {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func unwrapPath(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}
