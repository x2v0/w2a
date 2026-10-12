// Command wta-validatemodules validates modules/ catalog (Go port of
// .github/scripts/ci/validate_modules.py).
package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/modules"
)

func main() {
	repo := flag.String("repo", ".", "repository root")
	flag.Parse()

	abs, err := filepath.Abs(*repo)
	if err != nil {
		abs = *repo
	}
	os.Exit(modules.ValidateModulesDir(abs))
}
