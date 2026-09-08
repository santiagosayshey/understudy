// Command understudy stands in for Plex's image CDN so that actor portraits can
// be replaced. Each subcommand is one job; a container runs exactly one of them.
package main

import (
	"fmt"
	"os"
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
