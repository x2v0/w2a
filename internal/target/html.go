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
	// Resolve symlinks (WalkDir does not follow them)
	if resolved, err := filepath.EvalSymlinks(src); err == nil {
		src = resolved
	}
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
		// Skip build artifacts / source-only files to keep APK small.
		base := filepath.Base(rel)
		if strings.HasPrefix(rel, "node_modules/") ||
			strings.HasPrefix(rel, ".git/") ||
			strings.HasPrefix(rel, "build/") ||
			strings.Contains(rel, "/build/") ||
			base == "README.md" ||
			base == "wta.yaml" ||
			base == "Makefile" ||
			base == "icon.svg" ||
			(strings.HasPrefix(base, "icon-") && strings.HasSuffix(base, ".png")) ||
			strings.HasSuffix(base, ".apk") ||
			strings.HasSuffix(base, ".idsig") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dst := "assets/html/" + rel // shell reads from assets/html (siteAssetBase default)
		if err := a.AddAssetBytes(dst, data); err != nil {
			return err
		}
		n++
		return nil
	})
	if err != nil {
		return err
	}

	// ShellConfig expects htmlConfig.entryFile (ShellModeManager.kt):
	//   val entryFile = config.htmlConfig.entryFile
	//   entryFile.isNotBlank() && entryFile.substringBeforeLast(".").isNotBlank()
	cfgJSON := map[string]any{
		"appType":  "HTML",
		"language": "RUSSIAN",
		"htmlConfig": map[string]any{
			"entryFile": entry,
		},
	}
	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/app_config.json", data)

	fmt.Printf("html: %d files from %s\n", n, src)
	return a.SaveAPK(outPath)
}
