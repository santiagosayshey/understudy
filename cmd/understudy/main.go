// Command understudy stands in for Plex's image CDN so that actor portraits can
// be replaced. Each subcommand is one job; a container runs exactly one of them.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/santiagosayshey/understudy/internal/web"
)

// version is set by the linker at release time.
var version = "dev"

const usage = `usage: understudy <command> [flags]

  proxy     stand in for the CDN beside Plex
  sync      resolve people to their current portrait URLs and write the state
  validate  check the configuration against Plex without changing anything
  edit      serve the editor and its API
  cert      write the certificate authority and the leaf certificate
  version   print the version
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	args := os.Args[2:]
	switch os.Args[1] {
	case "edit":
		os.Exit(runEdit(args))
	case "validate":
		os.Exit(runValidate(args))
	case "sync":
		os.Exit(runSync(args))
	case "proxy":
		os.Exit(runProxy(args))
	case "cert":
		os.Exit(runCert(args))
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

// runEdit serves the embedded page and the editor's API. Only status exists so far.
func runEdit(args []string) int {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	listen := fs.String("listen", envOr("UNDERSTUDY_LISTEN", ":8090"), "address to serve the editor on")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": version})
	})
	mux.Handle("/", web.Handler())
	log.Printf("understudy %s: editor on %s", version, *listen)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.Print(err)
		return 1
	}
	return 0
}
