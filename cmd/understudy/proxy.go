package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/santiagosayshey/understudy/internal/plex"
	"github.com/santiagosayshey/understudy/internal/proxy"
)

// runProxy stands in for the CDN until told to stop.
func runProxy(args []string) int {
	fs := flag.NewFlagSet("proxy", flag.ContinueOnError)
	listen := fs.String("listen", envOr("UNDERSTUDY_LISTEN", ":443"), "address to serve TLS on")
	certDir := fs.String("certs", envOr("UNDERSTUDY_CERTS", "/certs"), "directory holding leaf.crt and leaf.key")
	cdn := fs.String("cdn", envOr("UNDERSTUDY_CDN", plex.CDN), "the real CDN to forward misses to")
	stateDir := fs.String("state", envOr("UNDERSTUDY_STATE", "/state"), "directory the resolving job writes the state file to")
	portraits := fs.String("portraits", envOr("UNDERSTUDY_PORTRAITS", "/portraits"), "directory the configuration's images live in")
	if err := fs.Parse(args); err != nil {
		return exitFailed
	}
	logger := log.New(os.Stdout, "", log.LstdFlags)
	p, err := proxy.New(proxy.Options{
		CertFile: filepath.Join(*certDir, "leaf.crt"), KeyFile: filepath.Join(*certDir, "leaf.key"),
		CDN: *cdn, StateDir: *stateDir, Portraits: *portraits, Logger: logger,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "proxy:", err)
		return exitFailed
	}
	l, err := net.Listen("tcp", *listen)
	if err != nil {
		fmt.Fprintln(os.Stderr, "proxy:", err)
		return exitFailed
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Printf("understudy %s: proxy on %s, forwarding misses to %s", version, l.Addr(), *cdn)
	if err := p.Serve(ctx, l); err != nil {
		fmt.Fprintln(os.Stderr, "proxy:", err)
		return exitFailed
	}
	return exitClean
}
