// Package plex is the read-only client for the parts of Plex Media Server's
// API that Understudy needs: which libraries exist, which actors each one
// knows, which titles an actor appears in, and the cast of one title. It is
// the only place a person's current portrait URL can be read from.
package plex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CDN is the host Plex fetches portraits from. Paths are relative to it.
const CDN = "https://metadata-static.plex.tv"

// Client talks to one server. The token is optional because the URL may be a
// proxy that injects it.
type Client struct {
	base  string
	token string
	http  *http.Client
}

func New(base, token string) *Client {
	return &Client{base: strings.TrimRight(base, "/"), token: token, http: &http.Client{Timeout: 90 * time.Second}}
}

// Section is a library. Only movie and show sections carry actors.
type Section struct {
	Key   string
	Type  string
	Title string
}

// Actor is one entry in a section's actor listing. Key is the server-local
// tag id, shared across sections. Thumb is the full portrait URL, or empty
// when Plex has no photo for the person.
type Actor struct {
	Key   string
	Name  string
	Thumb string
}

// Title is a movie or show an actor appears in.
type Title struct {
	RatingKey string
	Name      string
	Year      int
}

// Role is one cast entry on a title. TagKey is Plex's global person id, which
// only appears here and not in the actor listing.
type Role struct {
	Key    string
	Name   string
	TagKey string
	Thumb  string
	Role   string // the character, when Plex knows it
}

// Path turns a portrait URL into its CDN path. The second value is false for
// anything not on the CDN, such as the placeholder icon.
func Path(thumb string) (string, bool) {
	if !strings.HasPrefix(thumb, CDN+"/") {
		return "", false
	}
	return strings.TrimPrefix(thumb, CDN), true
}

// Sections lists the movie and show libraries.
func (c *Client) Sections(ctx context.Context) ([]Section, error) {
	var out struct {
		MediaContainer struct {
			Directory []struct {
				Key   str    `json:"key"`
				Type  string `json:"type"`
				Title string `json:"title"`
			}
		}
	}
	if err := c.get(ctx, "/library/sections", &out); err != nil {
		return nil, err
	}
	var sections []Section
	for _, d := range out.MediaContainer.Directory {
		if d.Type == "movie" || d.Type == "show" {
			sections = append(sections, Section{Key: string(d.Key), Type: d.Type, Title: d.Title})
		}
	}
	return sections, nil
}

// Actors lists every actor in a section. Plex offers no name filter on this
// endpoint, so callers index the whole listing.
func (c *Client) Actors(ctx context.Context, section string) ([]Actor, error) {
	var out struct {
		MediaContainer struct {
			Directory []struct {
				Key   str    `json:"key"`
				Title string `json:"title"`
				Thumb string `json:"thumb"`
			}
		}
	}
	if err := c.get(ctx, "/library/sections/"+url.PathEscape(section)+"/actor", &out); err != nil {
		return nil, err
	}
	actors := make([]Actor, 0, len(out.MediaContainer.Directory))
	for _, d := range out.MediaContainer.Directory {
		thumb := d.Thumb
		if _, ok := Path(thumb); !ok {
			thumb = ""
		}
		actors = append(actors, Actor{Key: string(d.Key), Name: d.Title, Thumb: thumb})
	}
	return actors, nil
}

// Titles lists what an actor appears in within one section.
func (c *Client) Titles(ctx context.Context, section, actor string) ([]Title, error) {
	var out struct {
		MediaContainer struct {
			Metadata []struct {
				RatingKey str    `json:"ratingKey"`
				Title     string `json:"title"`
				Year      int    `json:"year"`
			}
		}
	}
	p := "/library/sections/" + url.PathEscape(section) + "/all?actor=" + url.QueryEscape(actor)
	if err := c.get(ctx, p, &out); err != nil {
		return nil, err
	}
	titles := make([]Title, 0, len(out.MediaContainer.Metadata))
	for _, m := range out.MediaContainer.Metadata {
		titles = append(titles, Title{RatingKey: string(m.RatingKey), Name: m.Title, Year: m.Year})
	}
	return titles, nil
}

// Item is one movie or show with its cast.
type Item struct {
	RatingKey string
	Title     string
	Year      int
	Type      string
	Library   string
	Roles     []Role
}

// Item returns one title and its cast. The second result is false when
// Plex has no such title.
func (c *Client) Item(ctx context.Context, ratingKey string) (Item, bool, error) {
	var out struct {
		MediaContainer struct {
			Metadata []struct {
				RatingKey str    `json:"ratingKey"`
				Title     string `json:"title"`
				Year      int    `json:"year"`
				Type      string `json:"type"`
				Library   string `json:"librarySectionTitle"`
				Role      []struct {
					ID     str    `json:"id"`
					Tag    string `json:"tag"`
					TagKey string `json:"tagKey"`
					Thumb  string `json:"thumb"`
					Role   string `json:"role"`
				} `json:"Role"`
			}
		}
	}
	if err := c.get(ctx, "/library/metadata/"+url.PathEscape(ratingKey), &out); err != nil {
		return Item{}, false, err
	}
	if len(out.MediaContainer.Metadata) == 0 {
		return Item{}, false, nil
	}
	m := out.MediaContainer.Metadata[0]
	it := Item{RatingKey: string(m.RatingKey), Title: m.Title, Year: m.Year, Type: m.Type, Library: m.Library}
	for _, r := range m.Role {
		it.Roles = append(it.Roles, Role{Key: string(r.ID), Name: r.Tag, TagKey: r.TagKey, Thumb: r.Thumb, Role: r.Role})
	}
	return it, true, nil
}

// Roles returns the cast of one title.
func (c *Client) Roles(ctx context.Context, ratingKey string) ([]Role, error) {
	it, _, err := c.Item(ctx, ratingKey)
	return it.Roles, err
}

// Thumb returns a title's poster image as Plex stores it.
func (c *Client) Thumb(ctx context.Context, ratingKey string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/library/metadata/"+url.PathEscape(ratingKey)+"/thumb", nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("X-Plex-Token", c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plex: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plex: thumb %s returned %s", ratingKey, res.Status)
	}
	return io.ReadAll(res.Body)
}

func (c *Client) get(ctx context.Context, path string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("X-Plex-Token", c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("plex: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("plex: %s returned %s", path, res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(into); err != nil {
		return fmt.Errorf("plex: decoding %s: %w", path, err)
	}
	return nil
}

// str decodes a JSON string or number as a string. Plex mixes the two for
// ids: a section key is "1", a role id is 60525.
type str string

func (s *str) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch t := v.(type) {
	case string:
		*s = str(t)
	case float64:
		*s = str(fmt.Sprintf("%.0f", t))
	case nil:
		*s = ""
	default:
		return fmt.Errorf("unexpected id %v", v)
	}
	return nil
}
