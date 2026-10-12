// Package stack implements WebToApp "feature stack" injection: taking a
// bundle ZIP (DEX + manifest fragment + assets + res) produced by a
// feature-stacks/* Gradle module and grafting it into a base APK.
package stack

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Bundle is a parsed feature-stack ZIP.
type Bundle struct {
	Path      string
	Metadata  Metadata
	DexFiles  map[string][]byte // "dex/classes.dex" -> raw bytes
	Assets    map[string][]byte // "assets/foo.txt" -> bytes (rel path as-is)
	ResFiles  map[string][]byte // "res/mipmap/..." -> bytes
	StackARSC []byte            // "stack.arsc" (may be nil)
	Manifest  []byte            // "manifest.xml" (AXML fragment)
}

// Metadata is decoded from stack.json.
type Metadata struct {
	ID        string `json:"id"`
	Version   int    `json:"version"`
	PackageID string `json:"packageId"`
	ResPrefix string `json:"resPrefix"`
}

// ReadBundle loads and decodes a feature-stack ZIP.
func ReadBundle(path string) (*Bundle, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("stack: open %s: %w", path, err)
	}
	defer zr.Close()

	b := &Bundle{
		Path:     path,
		DexFiles: map[string][]byte{},
		Assets:   map[string][]byte{},
		ResFiles: map[string][]byte{},
	}

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}

		name := f.Name
		switch {
		case name == "stack.json":
			if err := json.Unmarshal(data, &b.Metadata); err != nil {
				return nil, fmt.Errorf("stack: parse stack.json: %w", err)
			}
		case name == "manifest.xml":
			b.Manifest = data
		case name == "stack.arsc":
			b.StackARSC = data
		case strings.HasPrefix(name, "dex/") && strings.HasSuffix(name, ".dex"):
			b.DexFiles[strings.TrimPrefix(name, "dex/")] = data
		case strings.HasPrefix(name, "assets/"):
			b.Assets[name] = data
		case strings.HasPrefix(name, "res/"):
			b.ResFiles[name] = data
		}
	}

	if b.Metadata.ID == "" {
		return nil, fmt.Errorf("stack: stack.json missing or has no id")
	}
	return b, nil
}

// DexNames returns sorted names of DEX files (usually "classes.dex",
// "classes2.dex", ...).
func (b *Bundle) DexNames() []string {
	out := make([]string, 0, len(b.DexFiles))
	for k := range b.DexFiles {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// WriteBundle serialises a Bundle back to disk (for tests / tooling).
func (b *Bundle) WriteBundle(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	put := func(name string, data []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}

	meta, _ := json.Marshal(b.Metadata)
	if err := put("stack.json", meta); err != nil {
		return err
	}
	for name, data := range b.DexFiles {
		if err := put("dex/"+name, data); err != nil {
			return err
		}
	}
	for name, data := range b.Assets {
		if err := put(name, data); err != nil {
			return err
		}
	}
	for name, data := range b.ResFiles {
		if err := put(name, data); err != nil {
			return err
		}
	}
	if len(b.StackARSC) > 0 {
		if err := put("stack.arsc", b.StackARSC); err != nil {
			return err
		}
	}
	if len(b.Manifest) > 0 {
		if err := put("manifest.xml", b.Manifest); err != nil {
			return err
		}
	}
	return zw.Close()
}
