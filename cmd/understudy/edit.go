package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/santiagosayshey/understudy/internal/api"
	"github.com/santiagosayshey/understudy/internal/plex"
	"github.com/santiagosayshey/understudy/internal/web"
)

// runEdit serves the editor: the embedded page and its API. The actor
// listing loads in the background, since it takes a while.
func runEdit(args []string) int {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	var c common
	c.bind(fs)
	listen := fs.String("listen", envOr("UNDERSTUDY_LISTEN", ":8090"), "address to serve the editor on")
	stateDir := fs.String("state", envOr("UNDERSTUDY_STATE", "/state"), "directory the resolving job writes the state file to")
	if err := fs.Parse(args); err != nil {
		return exitFailed
	}
	if c.plexURL == "" {
		fmt.Fprintln(os.Stderr, "edit: a Plex address is required (--plex-url or UNDERSTUDY_PLEX_URL)")
		return exitFailed
	}
	client := plex.New(c.plexURL, c.plexToken)
	listing := api.NewListing(client)
	go listing.Refresh(context.Background())
	srv := &api.Server{
		Version: version, Listing: listing, Images: api.NewImages(client), Staging: api.NewStaging(),
		Config: c.config, Portraits: c.portraits, StateDir: *stateDir,
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Handler())
	mux.Handle("/", web.Handler())
	log.Printf("understudy %s: editor on %s, Plex at %s", version, *listen, c.plexURL)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.Print(err)
		return exitFailed
	}
	return exitClean
}
