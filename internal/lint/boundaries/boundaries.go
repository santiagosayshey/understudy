// Package boundaries enforces the seams in the design as import rules, so
// the proxy cannot grow a dependency on Plex's API by accident and the Plex
// client cannot grow one on anything of ours. One rule per line in rules;
// add a line to add a rule.
package boundaries

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const module = "github.com/santiagosayshey/understudy/"

// rules maps a package (relative to the module) to the packages it must not
// import (also relative), with the reason shown in the report.
var rules = map[string][]forbidden{
	"internal/proxy": {
		{"internal/plex", "the proxy never talks to Plex's API; it reads the state file"},
		{"internal/api", "the proxy does not know about the editor"},
		{"internal/resolve", "resolving is the sync job's, not the proxy's"},
	},
	"internal/state": {
		{"internal/plex", "state is derived from resolve outcomes, never from Plex directly"},
		{"internal/api", "state does not know about the editor"},
	},
	"internal/face": {
		{"internal/", "the face detector depends on nothing of ours"},
	},
	"internal/plex": {
		{"internal/", "the Plex client depends on nothing of ours"},
	},
	"internal/config": {
		{"internal/plex", "the configuration is checked without Plex; resolving is separate"},
		{"internal/state", "the configuration is the input, state the output; they do not meet"},
	},
}

type forbidden struct {
	prefix string
	reason string
}

var Analyzer = &analysis.Analyzer{
	Name: "boundaries",
	Doc:  "reports imports that cross the seams in the design",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	pkg := strings.TrimSuffix(strings.TrimPrefix(pass.Pkg.Path(), module), "_test")
	var forbid []forbidden
	var roots []string
	for prefix, list := range rules {
		if pkg == prefix || strings.HasPrefix(pkg, prefix+"/") {
			forbid = append(forbid, list...)
			roots = append(roots, prefix)
		}
	}
	if len(forbid) == 0 {
		return nil, nil
	}
	within := func(rel string) bool { // inside the same rule root: a package and its subpackages
		for _, r := range roots {
			if rel == r || strings.HasPrefix(rel, r+"/") {
				return true
			}
		}
		return false
	}
	for _, f := range pass.Files {
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil || !strings.HasPrefix(path, module) {
				continue
			}
			rel := strings.TrimPrefix(path, module)
			for _, fb := range forbid {
				if rel == fb.prefix || strings.HasPrefix(rel, strings.TrimSuffix(fb.prefix, "/")+"/") {
					if within(rel) {
						continue
					}
					pass.Reportf(imp.Pos(), "%s must not import %s: %s", pkg, rel, fb.reason)
				}
			}
		}
	}
	return nil, nil
}

// Describe lists the rules, for the lint command's help.
func Describe() string {
	var b strings.Builder
	for pkg, list := range rules {
		for _, f := range list {
			fmt.Fprintf(&b, "  %s must not import %s (%s)\n", pkg, f.prefix, f.reason)
		}
	}
	return b.String()
}
