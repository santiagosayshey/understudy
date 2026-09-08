package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/santiagosayshey/understudy/internal/config"
	"github.com/santiagosayshey/understudy/internal/plex"
	"github.com/santiagosayshey/understudy/internal/resolve"
)

// Exit statuses shared by validate and sync: clean, finished with problems,
// could not finish. A pipeline shows the middle one as attention needed
// without hiding the log.
const (
	exitClean    = 0
	exitFailed   = 1
	exitProblems = 2
)

// runValidate checks the configuration against Plex and changes nothing.
func runValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	var c common
	c.bind(fs)
	if err := fs.Parse(args); err != nil {
		return exitFailed
	}
	if c.plexURL == "" {
		fmt.Fprintln(os.Stderr, "validate: a Plex address is required (--plex-url or UNDERSTUDY_PLEX_URL); the mistakes that matter are the ones only Plex can reveal")
		return exitFailed
	}
	cfg, err := config.Load(c.config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "validate:", err)
		return exitFailed
	}
	local := config.Validate(cfg, c.portraits)
	r := &resolve.Resolver{Plex: plex.New(c.plexURL, c.plexToken)}
	outcomes, err := r.Resolve(context.Background(), cfg.People)
	if err != nil {
		fmt.Fprintln(os.Stderr, "validate:", err)
		return exitFailed
	}
	return printReport(os.Stdout, cfg, local, outcomes)
}

// printReport writes one line per entry and a summary, and returns the exit
// status the lines add up to.
func printReport(w io.Writer, cfg *config.Config, local []config.Problem, outcomes []resolve.Outcome) int {
	width := 0
	for _, e := range cfg.People {
		width = max(width, len(e.Name))
	}
	var ok, warned, failed int
	for i, e := range cfg.People {
		type line struct{ label, text string }
		var lines []line
		problem := false
		for _, p := range local {
			if p.Index != i {
				continue
			}
			if p.Severity == config.Error {
				problem = true
				lines = append(lines, line{"problem", p.Detail})
			} else {
				lines = append(lines, line{"warning", p.Detail})
			}
		}
		o := outcomes[i]
		if o.Problem != nil {
			problem = true
			lines = append(lines, line{"problem", o.Problem.Detail})
		}
		for _, l := range lines {
			fmt.Fprintf(w, "%-8s %-*s  %s\n", l.label, width, e.Name, l.text)
		}
		switch {
		case problem:
			failed++
		case len(lines) > 0:
			warned++
			fmt.Fprintf(w, "%-8s %-*s  %s\n", "ok", width, e.Name, o.Person.Path)
		default:
			ok++
			fmt.Fprintf(w, "%-8s %-*s  %s\n", "ok", width, e.Name, o.Person.Path)
		}
	}
	fmt.Fprintf(w, "%d entries: %d ok, %d with warnings, %d with problems\n", len(cfg.People), ok, warned, failed)
	if failed > 0 {
		return exitProblems
	}
	return exitClean
}
