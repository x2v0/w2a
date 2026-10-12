// Command wta-gensubmissions generates modules/submissions.json
// (Go port of .github/scripts/ci/generate_submissions.py).
package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/modules"
)

func main() {
	owner := flag.String("owner", envOr("REPO_OWNER", "shiaho777"), "GitHub owner")
	repo := flag.String("repo", envOr("REPO_NAME", "web-to-app"), "GitHub repo")
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	maintainers := flag.String("maintainers", envOr("MAINTAINERS", ""), "comma-separated logins allowed to seed modules via direct push")
	offline := flag.Bool("offline", false, "skip GitHub API, use git metadata only")
	checkOnly := flag.Bool("check", false, "generate to temp and exit 1 if differs from committed file")
	root := flag.String("repo-root", ".", "repository root")
	flag.Parse()

	abs, err := filepath.Abs(*root)
	if err != nil {
		abs = *root
	}

	maintSet := map[string]struct{}{}
	for _, m := range strings.Split(*maintainers, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			maintSet[strings.ToLower(m)] = struct{}{}
		}
	}

	os.Exit(modules.GenerateSubmissions(
		abs, *owner, *repo, *token,
		maintSet, *offline, *checkOnly,
	))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
