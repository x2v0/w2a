package target

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
)

// FrontendTarget bundles a built SPA (React/Vue/Svelte dist/).
type FrontendTarget struct{}

func (t *FrontendTarget) Name() string { return "frontend" }

func (t *FrontendTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Frontend.Source)
	// Resolve symlinks (WalkDir does not follow them)
	if resolved, err := filepath.EvalSymlinks(src); err == nil {
		src = resolved
	}
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("frontend: source not found: %s", src)
	}
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}

	n := 0
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := a.AddAssetBytes("assets/html/"+filepath.ToSlash(rel), data); err != nil {
			return err
		}
		n++
		return nil
	})
	if err != nil {
		return err
	}

	// ShellConfig expects htmlConfig.entryFile (ShellModeManager.kt):
	//   normalizedAppType == "FRONTEND" -> config.htmlConfig.entryFile
	cfgJSON := map[string]any{
		"appType":  "FRONTEND",
		"language": "RUSSIAN",
		"htmlConfig": map[string]any{
			"entryFile": "index.html",
		},
	}
	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/app_config.json", data)

	fmt.Printf("frontend: %d files from %s\n", n, src)
	return a.SaveAPK(outPath)
}
