package target

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
)

// HTMLTarget bundles a local HTML project into assets/.
type HTMLTarget struct{}

func (t *HTMLTarget) Name() string { return "html" }

func (t *HTMLTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.HTML.Source)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("html: source not found: %s", src)
	}
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}

	entry := cfg.HTML.EntryFile
	if entry == "" {
		entry = "index.html"
	}

	n := 0
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		rel = filepath.ToSlash(rel)
		// Skip build artifacts / node_modules to keep APK small.
		if strings.HasPrefix(rel, "node_modules/") ||
			strings.HasPrefix(rel, ".git/") ||
			strings.HasPrefix(rel, "build/") && !strings.Contains(rel, "/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dst := "assets/webroot/" + rel
		if err := a.AddAssetBytes(dst, data); err != nil {
			return err
		}
		n++
		return nil
	})
	if err != nil {
		return err
	}

	cfgJSON := map[string]any{
		"appType":  "HTML",
		"entryUrl": "file:///android_asset/webroot/" + entry,
	}
	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/webtoapp/config.json", data)

	fmt.Printf("html: %d files from %s\n", n, src)
	return a.SaveAPK(outPath)
}
