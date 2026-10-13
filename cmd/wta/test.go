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
	OK       bool   `json:"ok"`
	Path     string `json:"path,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Duration int64  `json:"durationMs"`
	Error    string `json:"error,omitempty"`
}

func cmdTest(args []string) {
	// Pre-scan: extract --json anywhere in args
	// (Go's flag package stops at first positional arg)
	args, jsonOn := extractJSONFlag(args)

	fs := flag.NewFlagSet("test", flag.ExitOnError)
	asJSON := fs.Bool("json", jsonOn, "machine-readable JSON output")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: wta test <dir> [--json]")
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
		if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
			yamls = append(yamls, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(yamls)

	if len(yamls) == 0 {
		fmt.Fprintf(os.Stderr, "wta: no .yaml files in %s\n", dir)
		os.Exit(1)
	}

	// Заглушаем stdout build.Build — иначе шум
	realStdout := os.Stdout
	os.Stdout = os.Stderr
	defer func() { os.Stdout = realStdout }()

	var results []testResult
	passed, failed := 0, 0

	for _, y := range yamls {
		t0 := time.Now()
		cfg, err := config.Load(y)
		if err != nil {
			results = append(results, testResult{
				Name:     filepath.Base(y),
				OK:       false,
				Duration: time.Since(t0).Milliseconds(),
				Error:    err.Error(),
			})
			failed++
			continue
		}

		res, err := build.Build(cfg)
		dur := time.Since(t0).Milliseconds()

		if err != nil {
			results = append(results, testResult{
				Name: filepath.Base(y), Type: cfg.Type,
				OK: false, Duration: dur, Error: err.Error(),
			})
			failed++
			continue
		}

		results = append(results, testResult{
			Name: filepath.Base(y), Type: cfg.Type,
			OK: true, Path: res.OutputPath, Size: res.Size,
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
			"results": results,
		})
		if failed > 0 {
			os.Exit(1)
		}
		return
	}

	fmt.Println()
	for _, r := range results {
		mark := "✓"
		if !r.OK {
			mark = "✗"
		}
		if r.OK {
			fmt.Printf("  %s %-28s %-10s %6.2f MB  %5d ms\n",
				mark, r.Name, r.Type, float64(r.Size)/1024/1024, r.Duration)
		} else {
			errShort := r.Error
			if len(errShort) > 60 {
				errShort = errShort[:57] + "..."
			}
			fmt.Printf("  %s %-28s %-10s ERROR: %s\n",
				mark, r.Name, r.Type, errShort)
		}
	}
	fmt.Printf("\nResult: %d/%d passed, %d failed\n", passed, passed+failed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
