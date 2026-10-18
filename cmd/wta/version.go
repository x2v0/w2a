package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Overridable via:
//
//	go build -ldflags "-X main.Version=v1.2.3 \
//	                   -X main.Commit=$(git rev-parse --short HEAD) \
//	                   -X main.Built=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	Version = "0.1.0"
	Commit  = ""
	Built   = ""
)

func cmdVersion(args []string) {
	asJSON := false
	short := false
	for _, a := range args {
		switch a {
		case "--json", "-json":
			asJSON = true
		case "--short", "-s":
			short = true
		}
	}

	commit := Commit
	if commit == "" {
		if out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
			commit = strings.TrimSpace(string(out))
		}
	}

	if short {
		fmt.Println(Version)
		return
	}

	if asJSON {
		out := map[string]string{
			"version": Version,
			"commit":  commit,
			"go":      runtime.Version(),
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
			"built":   Built,
		}
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
		return
	}

	fmt.Printf("wta %s", Version)
	if commit != "" {
		fmt.Printf(" (%s)", commit)
	}
	fmt.Println()
	fmt.Printf("  go:    %s\n", runtime.Version())
	fmt.Printf("  os:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	if Built != "" {
		fmt.Printf("  built: %s\n", Built)
	}
}
