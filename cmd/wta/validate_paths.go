package main

import (
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/config"
)

// sourceDir returns the local source directory for the config type.
// Returns "" for types that have no local source (web, multiweb, wordpress).
func sourceDir(cfg *config.Config) string {
	switch cfg.Type {
	case "html":
		if cfg.HTML != nil {
			return cfg.HTML.Source
		}
	case "frontend":
		if cfg.Frontend != nil {
			return cfg.Frontend.Source
		}
	case "nodejs":
		if cfg.NodeJS != nil {
			return cfg.NodeJS.Source
		}
	}
	// TODO: extend for go, python, php, gallery, image, video
	// once their Section types are confirmed.
	return ""
}

// validateSourceOutput returns a warning if the output APK path is inside
// the source directory. In that case WalkDir on next build may pick up its
// own artifacts (APK, .idsig, logs) — recursive self-inclusion that doubles
// APK size (observed: 81 MB → 162 MB).
//
// Returns "" if OK.
func validateSourceOutput(cfg *config.Config) string {
	src := sourceDir(cfg)
	if src == "" {
		return ""
	}
	out := cfg.Output.Path
	if out == "" {
		return ""
	}

	srcFull := cfg.Resolve(src)
	outFull := cfg.Resolve(out)
	if abs, err := filepath.Abs(srcFull); err == nil {
		srcFull = abs
	}
	if abs, err := filepath.Abs(outFull); err == nil {
		outFull = abs
	}

	// If output looks like a file, use its parent dir.
	outDir := outFull
	if ext := filepath.Ext(outFull); ext != "" {
		outDir = filepath.Dir(outFull)
	}

	rel, err := filepath.Rel(srcFull, outDir)
	if err != nil {
		return ""
	}
	// Outside source: rel starts with ".." — OK.
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return "output path is inside source dir; build artifacts may be re-included " +
		"on next build — move it outside: output.path: ../<name>.apk"
}
