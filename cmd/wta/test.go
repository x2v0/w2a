package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/x2v0/w2a/go/internal/build"
	"github.com/x2v0/w2a/go/internal/config"
)

type testResult struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status"` // ok | fail | skip
	Path     string `json:"path,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Duration int64  `json:"durationMs"`
	Error    string `json:"error,omitempty"`
}

// skipReason returns non-empty string if the config's source path is
// missing (then build would fail anyway). Lets "wta test" show ⊘ skipped
// instead of ✗ error for missing fixtures.
func skipReason(cfg *config.Config) string {
	var src string
	switch cfg.Type {
	case "html":
		if cfg.HTML != nil {
			src = cfg.HTML.Source
		}
	case "frontend":
		if cfg.Frontend != nil {
			src = cfg.Frontend.Source
		}
	case "go":
		if cfg.Go != nil {
			src = cfg.Go.Source
		}
	case "gallery":
		if cfg.Gallery != nil {
			src = cfg.Gallery.Source
		}
	case "image":
		if cfg.Image != nil {
			src = cfg.Image.Source
		}
	case "video":
		if cfg.Video != nil {
			src = cfg.Video.Source
		}
	default:
		return ""
	}
	if src == "" {
		return ""
	}
	full := cfg.Resolve(src)
	if _, err := os.Stat(full); err != nil {
		return "source not found: " + src
	}
	return ""
}

func cmdTest(args []string) {
	// Pre-scan: extract --json and --filter anywhere in args
	var jsonOn, listOn bool
	var filterStr string
	args, jsonOn, filterStr, listOn = extractTestFlags(args)

	fs := flag.NewFlagSet("test", flag.ExitOnError)
	asJSON := fs.Bool("json", jsonOn, "machine-readable JSON output")
	filter := fs.String("filter", filterStr, "only yaml files matching substring")
	listOnly := fs.Bool("list", listOn, "list yaml files without building")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: wta test <dir> [--json] [--filter <s>]")
		os.Exit(2)
	}
	dir := fs.Arg(0)

	entries, err := os.ReadDir(dir)
	if err != nil {
		die(err)
	}

	var yamls []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".yaml") && !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		if *filter != "" && !strings.Contains(e.Name(), *filter) {
			continue
		}
		yamls = append(yamls, filepath.Join(dir, e.Name()))
	}
	sort.Strings(yamls)

	if *listOnly {
		for _, y := range yamls {
			fmt.Println(filepath.Base(y))
		}
		return
	}

	if len(yamls) == 0 {
		fmt.Fprintf(os.Stderr, "wta: no .yaml files in %s\n", dir)
		os.Exit(1)
	}

	// Заглушаем stdout build.Build — иначе шум
	realStdout := os.Stdout
	os.Stdout = os.Stderr
	defer func() { os.Stdout = realStdout }()

	var results []testResult
	passed, failed, skipped := 0, 0, 0

	for _, y := range yamls {
		t0 := time.Now()
		cfg, err := config.Load(y)
		if err != nil {
			results = append(results, testResult{
				Name:     filepath.Base(y),
				Status:   "fail",
				Duration: time.Since(t0).Milliseconds(),
				Error:    err.Error(),
			})
			failed++
			continue
		}

		if reason := skipReason(cfg); reason != "" {
			results = append(results, testResult{
				Name:     filepath.Base(y),
				Type:     cfg.Type,
				Status:   "skip",
				Duration: time.Since(t0).Milliseconds(),
				Error:    reason,
			})
			skipped++
			continue
		}

		res, err := build.Build(cfg)
		dur := time.Since(t0).Milliseconds()

		if err != nil {
			results = append(results, testResult{
				Name: filepath.Base(y), Type: cfg.Type,
				Status: "fail", Duration: dur, Error: err.Error(),
			})
			failed++
			continue
		}

		results = append(results, testResult{
			Name: filepath.Base(y), Type: cfg.Type,
			Status: "ok", Path: res.OutputPath, Size: res.Size,
			Duration: dur,
		})
		passed++
	}

	os.Stdout = realStdout

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{
			"ok":      failed == 0,
			"passed":  passed,
			"failed":  failed,
			"skipped": skipped,
			"results": results,
		})
		if failed > 0 {
			os.Exit(1)
		}
		return
	}

	fmt.Println()
	for _, r := range results {
		switch r.Status {
		case "ok":
			fmt.Printf("  ✓ %-28s %-10s %6.2f MB  %5d ms\n",
				r.Name, r.Type, float64(r.Size)/1024/1024, r.Duration)
		case "skip":
			fmt.Printf("  ⊘ %-28s %-10s SKIP: %s\n",
				r.Name, r.Type, r.Error)
		default:
			errShort := r.Error
			if len(errShort) > 60 {
				errShort = errShort[:57] + "..."
			}
			fmt.Printf("  ✗ %-28s %-10s ERROR: %s\n",
				r.Name, r.Type, errShort)
		}
	}
	fmt.Printf("\nResult: %d passed, %d failed, %d skipped (total %d)\n",
		passed, failed, skipped, passed+failed+skipped)
	if failed > 0 {
		os.Exit(1)
	}
}
