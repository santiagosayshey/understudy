// Command lint runs Understudy's own Go analyzers, the rules that go vet and
// staticcheck do not know about. Run it as `go run ./cmd/lint ./...`.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/santiagosayshey/understudy/internal/lint/boundaries"
)

func main() {
	multichecker.Main(
		boundaries.Analyzer,
	)
}
