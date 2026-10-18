package main

import (
	"archive/zip"
	"fmt"
	"sort"
	"strings"
)

// treeNode is a single node (file or directory) in the APK tree.
type treeNode struct {
	name     string
	size     uint64
	isFile   bool
	children map[string]*treeNode
}

func buildTree(files []*zip.File) *treeNode {
	root := &treeNode{children: map[string]*treeNode{}}
	for _, f := range files {
		parts := strings.Split(f.Name, "/")
		cur := root
		for i, p := range parts {
			if p == "" {
				continue
			}
			if cur.children == nil {
				cur.children = map[string]*treeNode{}
			}
			child, ok := cur.children[p]
			if !ok {
				child = &treeNode{name: p, children: map[string]*treeNode{}}
				cur.children[p] = child
			}
			if i == len(parts)-1 {
				child.isFile = true
				child.size = f.UncompressedSize64
			}
			cur = child
		}
	}
	return root
}

// renderTree prints children of root using box-drawing characters.
func renderTree(root *treeNode, prefix string) {
	names := make([]string, 0, len(root.children))
	for n := range root.children {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := root.children[names[i]], root.children[names[j]]
		if a.isFile != b.isFile {
			return !a.isFile // dirs first
		}
		return names[i] < names[j]
	})

	for i, n := range names {
		child := root.children[n]
		last := i == len(names)-1
		branch := "├── "
		nextPrefix := prefix + "│   "
		if last {
			branch = "└── "
			nextPrefix = prefix + "    "
		}
		suffix := "/"
		if child.isFile {
			suffix = "  " + humanBytes(child.size)
		}
		fmt.Printf("%s%s%s%s\n", prefix, branch, n, suffix)
		if !child.isFile {
			renderTree(child, nextPrefix)
		}
	}
}

// sizeCategory classifies an APK entry for size breakdown.
func sizeCategory(name string) string {
	switch {
	case name == "AndroidManifest.xml":
		return "manifest"
	case strings.HasPrefix(name, "lib/"):
		return "native libs"
	case strings.HasPrefix(name, "classes") && strings.HasSuffix(name, ".dex"):
		return "dex"
	case name == "resources.arsc" || strings.HasPrefix(name, "res/"):
		return "resources"
	case strings.HasPrefix(name, "assets/"):
		return "assets"
	case strings.HasPrefix(name, "META-INF/"):
		return "META-INF"
	case strings.HasPrefix(name, "kotlin/"):
		return "kotlin"
	default:
		return "other"
	}
}

func printSize(zr *zip.Reader) {
	cats := map[string]uint64{}
	var total uint64
	for _, f := range zr.File {
		cats[sizeCategory(f.Name)] += f.UncompressedSize64
		total += f.UncompressedSize64
	}

	type kv struct {
		name string
		size uint64
	}
	var ordered []kv
	for n, sz := range cats {
		ordered = append(ordered, kv{n, sz})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].size > ordered[j].size })

	fmt.Printf("Summary (%s total, uncompressed)\n", humanBytes(total))
	for _, o := range ordered {
		pct := 0.0
		if total > 0 {
			pct = float64(o.size) * 100 / float64(total)
		}
		fmt.Printf("  %-12s %10s  %5.1f%%\n", o.name, humanBytes(o.size), pct)
	}

	// per-ABI breakdown for lib/
	abis := map[string]uint64{}
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "lib/") {
			parts := strings.Split(f.Name, "/")
			if len(parts) >= 3 {
				abis[parts[1]] += f.UncompressedSize64
			}
		}
	}
	if len(abis) > 0 {
		fmt.Println()
		fmt.Println("Native libs per ABI:")
		var ord []kv
		for n, sz := range abis {
			ord = append(ord, kv{n, sz})
		}
		sort.Slice(ord, func(i, j int) bool { return ord[i].size > ord[j].size })
		for _, o := range ord {
			fmt.Printf("  %-15s %10s\n", o.name, humanBytes(o.size))
		}
	}
}

func humanBytes(n uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case n >= GB:
		return fmt.Sprintf("%.2f GB", float64(n)/float64(GB))
	case n >= MB:
		return fmt.Sprintf("%.2f MB", float64(n)/float64(MB))
	case n >= KB:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(KB))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
