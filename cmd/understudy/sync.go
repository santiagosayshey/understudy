package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/santiagosayshey/understudy/internal/config"
	"github.com/santiagosayshey/understudy/internal/plex"
	"github.com/santiagosayshey/understudy/internal/resolve"
	"github.com/santiagosayshey/understudy/internal/state"
)

// runSync resolves the configuration against Plex and writes the state file
// the proxy serves from. It validates first and stops on any error, so a
// broken configuration never reaches the state.
func runSync(args []string) int {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	var c common
	c.bind(fs)
	stateDir := fs.String("state", envOr("UNDERSTUDY_STATE", "/state"), "directory the state file is written to")
	cacheDir := fs.String("plex-cache", envOr("UNDERSTUDY_PLEX_CACHE", ""), "Plex's Cache/PhotoTranscoder directory, cleared when something changed; nothing is cleared without it")
	if err := fs.Parse(args); err != nil {
		return exitFailed
	}
	if c.plexURL == "" {
		fmt.Fprintln(os.Stderr, "sync: a Plex address is required (--plex-url or UNDERSTUDY_PLEX_URL)")
		return exitFailed
	}
	cfg, err := config.Load(c.config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		return exitFailed
	}
	local := config.Validate(cfg, c.portraits)
	if errs := errorsIn(local); len(errs) > 0 {
		for _, p := range errs {
			fmt.Fprintf(os.Stderr, "problem  %s  %s\n", cfg.People[p.Index].Name, p.Detail)
		}
		fmt.Fprintln(os.Stderr, "sync: the configuration has problems; nothing written")
		return exitFailed
	}
	prev, err := state.Load(*stateDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		return exitFailed
	}
	r := &resolve.Resolver{Plex: plex.New(c.plexURL, c.plexToken)}
	outcomes, err := r.Resolve(context.Background(), cfg.People)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		return exitFailed
	}
	hashes, err := imageHashes(cfg, c.portraits)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		return exitFailed
	}
	next, changes := state.Apply(prev, outcomes, hashes, time.Now())
	var cleared int
	if changes.Any() && *cacheDir != "" {
		n, err := plex.ClearCache(*cacheDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "sync:", err)
			return exitFailed
		}
		cleared, next.CacheCleared = n, true
	}
	if err := next.Save(*stateDir); err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		return exitFailed
	}
	status := printReport(os.Stdout, cfg, local, outcomes)
	for _, d := range changes.Drifted {
		fmt.Printf("drift    %s  %s -> %s\n", d.Name, d.From, d.To)
	}
	for _, n := range changes.Updated {
		fmt.Printf("updated  %s  image changed\n", n)
	}
	fmt.Printf("state: %d added, %d removed, %d updated, %d drifted; written to %s\n",
		len(changes.Added), len(changes.Removed), len(changes.Updated), len(changes.Drifted), filepath.Join(*stateDir, state.FileName))
	switch {
	case next.CacheCleared:
		fmt.Printf("cache: cleared %d entries from %s; clients pick up the change on their next fetch\n", cleared, *cacheDir)
	case changes.Any():
		fmt.Println("cache: not configured, nothing cleared; Plex keeps showing its cached portraits until its photo cache is cleared")
	default:
		fmt.Println("cache: nothing changed, nothing cleared")
	}
	return status
}

// imageHashes fingerprints every configured image so a replaced file under
// the same name counts as a change.
func imageHashes(cfg *config.Config, portraits string) (map[string]string, error) {
	hashes := map[string]string{}
	for _, e := range cfg.People {
		file, ok := config.ImagePath(portraits, e.Image)
		if !ok {
			continue
		}
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return nil, err
		}
		hashes[e.Image] = hex.EncodeToString(h.Sum(nil))
	}
	return hashes, nil
}

func errorsIn(problems []config.Problem) []config.Problem {
	var out []config.Problem
	for _, p := range problems {
		if p.Severity == config.Error {
			out = append(out, p)
		}
	}
	return out
}
