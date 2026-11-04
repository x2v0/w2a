package main

import (
	"fmt"

	"github.com/x2v0/w2a/go/internal/target"
)

type appTypeInfo struct {
	Name       string
	Impl       bool     // true — есть target handler
	SourceKind string   // "url" | "dir" | "file" | "sites"
	Fields     []string // yaml ключи специфичные
	Notes      string
}

var appTypes = []appTypeInfo{
	{"web", true, "url", []string{"url", "userAgent"}, "WebView wrapper for a URL"},
	{"html", true, "dir", []string{"source", "entryFile"}, "local HTML bundle in WebView"},
	{"frontend", true, "dir", []string{"source", "entryFile"}, "SPA / static build"},
	{"go", true, "dir", []string{"source", "abis", "noCgo"}, "cross-compiled to .so"},
	{"gallery", true, "dir", []string{"source", "columns"}, "image grid"},
	{"image", true, "file", []string{"source", "aspectRatio"}, "single image viewer"},
	{"video", true, "file", []string{"source", "autoplay"}, "video player"},
	{"multiweb", true, "sites", []string{"sites"}, "tabbed multi-site"},
	{"custom", true, "dir", []string{"sources", "jniLibs", "manifest"}, "Java + jniLibs (no gradle)"},
	{"nodejs", false, "dir", []string{"source", "entryModule", "port", "nodeVersion"}, "nodejs-mobile runtime, libnode.so per ABI"},
	{"python", false, "dir", []string{"source"}, "stub — not implemented"},
}

func cmdTypes(args []string) {
	for _, t := range appTypes {
		impl := t.Impl
		if !impl {
			if _, err := target.For(t.Name); err == nil {
				impl = true
			}
		}
		mark := "✓"
		notes := t.Notes
		if !impl {
			mark = "·"
		} else if notes == "stub — not implemented" {
			notes = ""
		}
		fmt.Printf("  %s %-10s  %-6s  %s\n", mark, t.Name, t.SourceKind, notes)
	}
}
