package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/santiagosayshey/understudy/internal/certs"
)

// runCert writes the certificate authority and the leaf certificate. The
// authority goes to the Plex container, the leaf pair stays with the proxy.
func runCert(args []string) int {
	fs := flag.NewFlagSet("cert", flag.ContinueOnError)
	out := fs.String("out", envOr("UNDERSTUDY_CERTS", "/certs"), "directory to write ca.crt, ca.key, leaf.crt and leaf.key into")
	hostname := fs.String("hostname", "metadata-static.plex.tv", "the CDN hostname the leaf is issued for")
	if err := fs.Parse(args); err != nil {
		return exitFailed
	}
	if err := certs.Generate(*out, *hostname); err != nil {
		fmt.Fprintln(os.Stderr, "cert:", err)
		return exitFailed
	}
	fmt.Printf("wrote %s: ca.crt (install this in the Plex container), ca.key (keep secret), leaf.crt and leaf.key (the proxy's)\n", *out)
	return exitClean
}
