package main

import (
	"context"
	"flag"
	"fmt"
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
	next, changes := state.Apply(prev, outcomes, time.Now())
	if err := next.Save(*stateDir); err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		return exitFailed
	}
	status := printReport(os.Stdout, cfg, local, outcomes)
	for _, d := range changes.Drifted {
		fmt.Printf("drift    %s  %s -> %s\n", d.Name, d.From, d.To)
	}
	fmt.Printf("state: %d added, %d removed, %d drifted; written to %s\n",
		len(changes.Added), len(changes.Removed), len(changes.Drifted), filepath.Join(*stateDir, state.FileName))
	return status
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
