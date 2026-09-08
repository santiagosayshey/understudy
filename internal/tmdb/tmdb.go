// Package tmdb is a small client for The Movie Database: search for people
// by name, read a person with their profile images, and fetch an image. It
// is where the editor's portraits come from when a key is set, and it
// depends on nothing of ours.
package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// API is the JSON API's base.
	API = "https://api.themoviedb.org"
	// Images is the image server's base; a size and a file path follow it.
	Images = "https://image.tmdb.org/t/p"
	// MaxImage caps a fetched image, the same cap the editor puts on an
	// upload.
	MaxImage = 40 << 20
)

// Client talks to TMDb with one key. Both a v3 API key and a v4 read access
// token work; the token is a JWT and is told apart by its dots.
type Client struct {
	API    string // overridden in tests
	Images string // overridden in tests
	key    string
	http   *http.Client
}

func New(key string) *Client {
	return &Client{API: API, Images: Images, key: key, http: &http.Client{Timeout: 30 * time.Second}}
}

// Person is a search result: enough to tell people of the same name apart.
type Person struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Profile    string   `json:"profile,omitempty"` // file path of the main profile image
	KnownFor   []string `json:"knownFor"`          // titles TMDb lists them for
	Popularity float64  `json:"-"`
}

// Profile is one of a person's images.
type Profile struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Detail is a person with their images and their IMDb id.
type Detail struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	IMDbID   string    `json:"imdbId,omitempty"`
	Profiles []Profile `json:"profiles"`
}

// Search finds people by name, in TMDb's order.
func (c *Client) Search(ctx context.Context, name string) ([]Person, error) {
	var out struct {
		Results []struct {
			ID          int     `json:"id"`
			Name        string  `json:"name"`
			ProfilePath string  `json:"profile_path"`
			Popularity  float64 `json:"popularity"`
			KnownFor    []struct {
				Title string `json:"title"` // movies
				Name  string `json:"name"`  // shows
			} `json:"known_for"`
		} `json:"results"`
	}
	if err := c.get(ctx, "/3/search/person", url.Values{"query": {name}}, &out); err != nil {
		return nil, err
	}
	people := make([]Person, 0, len(out.Results))
	for _, r := range out.Results {
		p := Person{ID: r.ID, Name: r.Name, Profile: r.ProfilePath, Popularity: r.Popularity, KnownFor: []string{}}
		for _, k := range r.KnownFor {
			if t := cmpOr(k.Title, k.Name); t != "" {
				p.KnownFor = append(p.KnownFor, t)
			}
		}
		people = append(people, p)
	}
	return people, nil
}

// Person reads one person with every profile image TMDb has for them, in
// TMDb's order, which puts the best voted first.
func (c *Client) Person(ctx context.Context, id int) (Detail, error) {
	var out struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		IMDbID string `json:"imdb_id"`
		Images struct {
			Profiles []struct {
				FilePath string `json:"file_path"`
				Width    int    `json:"width"`
				Height   int    `json:"height"`
			} `json:"profiles"`
		} `json:"images"`
	}
	path := "/3/person/" + strconv.Itoa(id)
	if err := c.get(ctx, path, url.Values{"append_to_response": {"images"}}, &out); err != nil {
		return Detail{}, err
	}
	d := Detail{ID: out.ID, Name: out.Name, IMDbID: out.IMDbID, Profiles: []Profile{}}
	for _, p := range out.Images.Profiles {
		d.Profiles = append(d.Profiles, Profile{Path: p.FilePath, Width: p.Width, Height: p.Height})
	}
	return d, nil
}

// Image fetches a file path at a size: w185, h632 or original for profiles.
func (c *Client) Image(ctx context.Context, path, size string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Images+"/"+size+path, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tmdb: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb: image %s returned %s", path, res.Status)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, MaxImage+1))
	if err != nil {
		return nil, fmt.Errorf("tmdb: %w", err)
	}
	if len(data) > MaxImage {
		return nil, fmt.Errorf("tmdb: image %s is over %d MB", path, MaxImage>>20)
	}
	return data, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, into any) error {
	bearer := strings.Contains(c.key, ".")
	if !bearer {
		q.Set("api_key", c.key)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.API+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if bearer {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("tmdb: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		// TMDb says why in the body: an invalid key, most likely.
		var why struct {
			Message string `json:"status_message"`
		}
		if json.NewDecoder(res.Body).Decode(&why) == nil && why.Message != "" {
			return fmt.Errorf("tmdb: %s returned %s: %s", path, res.Status, why.Message)
		}
		return fmt.Errorf("tmdb: %s returned %s", path, res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(into); err != nil {
		return fmt.Errorf("tmdb: decoding %s: %w", path, err)
	}
	return nil
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
