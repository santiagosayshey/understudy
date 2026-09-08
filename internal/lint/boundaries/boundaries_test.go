package boundaries

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// The analyzer is checked with hand-built passes rather than analysistest,
// which would need a second module on disk: what matters is which import
// paths are reported for which package.
func TestBoundaries(t *testing.T) {
	cases := []struct {
		pkg     string
		imports []string
		want    int
	}{
		{"internal/proxy", []string{"github.com/santiagosayshey/understudy/internal/state", "net/http"}, 0},
		{"internal/proxy", []string{"github.com/santiagosayshey/understudy/internal/plex"}, 1},
		{"internal/proxy", []string{"github.com/santiagosayshey/understudy/internal/api"}, 1},
		{"internal/plex", []string{"github.com/santiagosayshey/understudy/internal/config"}, 1},
		{"internal/plex", []string{"github.com/santiagosayshey/understudy/internal/plex/plextest"}, 0},
		{"internal/plex/plextest", []string{"github.com/santiagosayshey/understudy/internal/plex"}, 0},
		{"internal/state", []string{"github.com/santiagosayshey/understudy/internal/resolve"}, 0},
		{"internal/state", []string{"github.com/santiagosayshey/understudy/internal/plex"}, 1},
		{"internal/api", []string{"github.com/santiagosayshey/understudy/internal/plex"}, 0},
		{"cmd/understudy", []string{"github.com/santiagosayshey/understudy/internal/plex"}, 0},
	}
	for _, c := range cases {
		src := "package p\n"
		for _, imp := range c.imports {
			src += "import _ \"" + imp + "\"\n"
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "p.go", src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		var got int
		pass := &analysis.Pass{
			Analyzer: Analyzer,
			Fset:     fset,
			Files:    []*ast.File{f},
			Pkg:      types.NewPackage(module+c.pkg, "p"),
			Report:   func(analysis.Diagnostic) { got++ },
		}
		if _, err := run(pass); err != nil {
			t.Fatal(err)
		}
		if got != c.want {
			t.Errorf("%s importing %v: want %d reports, got %d", c.pkg, c.imports, c.want, got)
		}
	}
}
