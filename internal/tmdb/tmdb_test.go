package tmdb_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/santiagosayshey/understudy/internal/tmdb"
)

// fake answers the three calls the client makes and records how the key
// arrived, so both ways of authenticating are checked.
func fake(t *testing.T, big int) (*httptest.Server, *string) {
	t.Helper()
	var auth string
	mux := http.NewServeMux()
	mux.HandleFunc("/3/search/person", func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization") + "|" + r.URL.Query().Get("api_key")
		if r.URL.Query().Get("query") != "Cailee Spaeny" {
			w.Write([]byte(`{"results":[]}`))
			return
		}
		w.Write([]byte(`{"results":[
			{"id":1,"name":"Cailee Spaeny","profile_path":"/c.jpg","popularity":12.5,
			 "known_for":[{"title":"Priscilla"},{"name":"Mare of Easttown"},{}]},
			{"id":2,"name":"Cailee Spaeny","popularity":0.5,"known_for":[]}]}`))
	})
	mux.HandleFunc("/3/person/1", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("append_to_response") != "images" {
			http.Error(w, "no images asked for", http.StatusBadRequest)
			return
		}
		w.Write([]byte(`{"id":1,"name":"Cailee Spaeny","imdb_id":"nm8091257",
			"images":{"profiles":[{"file_path":"/a.jpg","width":1000,"height":1500},{"file_path":"/b.jpg","width":600,"height":900}]}}`))
	})
	mux.HandleFunc("/3/person/9", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status_code":7,"status_message":"Invalid API key: You must be granted a valid key."}`))
	})
	mux.HandleFunc("/original/a.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.Repeat([]byte{1}, 10))
	})
	mux.HandleFunc("/original/big.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.Repeat([]byte{1}, big))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &auth
}

func client(srv *httptest.Server, key string) *tmdb.Client {
	c := tmdb.New(key)
	c.API, c.Images = srv.URL, srv.URL
	return c
}

func TestClient(t *testing.T) {
	srv, auth := fake(t, tmdb.MaxImage+1)
	ctx := context.Background()
	c := client(srv, "v3key")

	people, err := c.Search(ctx, "Cailee Spaeny")
	if err != nil {
		t.Fatal(err)
	}
	if *auth != "|v3key" {
		t.Errorf("a v3 key goes in the query: %q", *auth)
	}
	if len(people) != 2 || people[0].ID != 1 || people[0].Profile != "/c.jpg" || people[0].Popularity != 12.5 {
		t.Fatalf("search: %+v", people)
	}
	if got := strings.Join(people[0].KnownFor, ","); got != "Priscilla,Mare of Easttown" {
		t.Errorf("known for takes a movie's title or a show's name, skipping neither: %q", got)
	}
	if people[1].KnownFor == nil {
		t.Error("known for is a list even when empty, for the JSON")
	}
	if none, err := c.Search(ctx, "Nobody"); err != nil || len(none) != 0 {
		t.Errorf("no results: %v %v", none, err)
	}

	d, err := c.Person(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if d.IMDbID != "nm8091257" || len(d.Profiles) != 2 || d.Profiles[0].Path != "/a.jpg" || d.Profiles[0].Width != 1000 {
		t.Errorf("person: %+v", d)
	}

	if _, err := c.Person(ctx, 9); err == nil || !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("TMDb's reason should be in the error: %v", err)
	}

	data, err := c.Image(ctx, "/a.jpg", "original")
	if err != nil || len(data) != 10 {
		t.Errorf("image: %d bytes, %v", len(data), err)
	}
	if _, err := c.Image(ctx, "/big.jpg", "original"); err == nil || !strings.Contains(err.Error(), "over 40 MB") {
		t.Errorf("an oversized image is refused: %v", err)
	}
	if _, err := c.Image(ctx, "/missing.jpg", "original"); err == nil {
		t.Error("a missing image is an error")
	}

	c = client(srv, "eyJhbGciOiJIUzI1NiJ9.token.sig")
	if _, err := c.Search(ctx, "Nobody"); err != nil {
		t.Fatal(err)
	}
	if *auth != "Bearer eyJhbGciOiJIUzI1NiJ9.token.sig|" {
		t.Errorf("a v4 token goes in the header: %q", *auth)
	}
}
