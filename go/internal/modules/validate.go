package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ValidateModulesDir is the CLI entry point for `wta-validatemodules`.
// Returns 0 on success, 1 on failure.
func ValidateModulesDir(repoRoot string) int {
	modulesDir := filepath.Join(repoRoot, "modules")
	info, err := os.Stat(modulesDir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "❌ %s does not exist\n", modulesDir)
		return 1
	}

	r := &Report{}

	// 1. Parse + validate registry.
	var registryEntries []map[string]any
	if registry := LoadJSON(r, filepath.Join(modulesDir, "registry.json"), "registry.json"); registry != nil {
		registryEntries = ValidateRegistry(r, registry)
	}

	// 2. Walk module folders (skip dotdirs).
	folders, err := os.ReadDir(modulesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ cannot read %s: %v\n", modulesDir, err)
		return 1
	}
	var folderPaths []string
	folderNames := map[string]struct{}{}
	for _, e := range folders {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		folderPaths = append(folderPaths, filepath.Join(modulesDir, e.Name()))
		folderNames[e.Name()] = struct{}{}
	}
	sort.Strings(folderPaths)

	// Registry paths split into: all, custom-only.
	registryPaths := map[string]struct{}{}
	customPaths := map[string]struct{}{}
	entriesByPath := map[string]map[string]any{}
	for _, e := range registryEntries {
		p, ok := e["path"].(string)
		if !ok || p == "" {
			continue
		}
		registryPaths[p] = struct{}{}
		sourceType := "CUSTOM"
		if s, ok := e["sourceType"].(string); ok && s != "" {
			sourceType = s
		}
		if sourceType != "CHROME_EXTENSION" {
			customPaths[p] = struct{}{}
			entriesByPath[p] = e
		}
	}

	// Folder present but no registry entry.
	for name := range folderNames {
		if _, ok := registryPaths[name]; !ok {
			r.Error("modules/"+name, "module folder has no entry in registry.json")
		}
	}
	// Registry entry points at a missing folder (custom only).
	for p := range customPaths {
		if _, ok := folderNames[p]; !ok {
			r.Error("registry.json", fmt.Sprintf("entry refers to missing folder modules/%s/", p))
		}
	}

	// 3. Per-folder validation + consistency with registry.
	for _, folder := range folderPaths {
		name := filepath.Base(folder)
		ValidateFolderLayout(r, folder)
		ValidateMainJs(r, folder)
		ValidateIconCoherence(r, folder, entriesByPath[name])

		pluginPath := filepath.Join(folder, "plugin.json")
		if plugin := LoadJSON(r, pluginPath, "modules/"+name+"/plugin.json"); plugin != nil {
			ValidatePluginJSON(r, name, plugin)
			if entry, ok := entriesByPath[name]; ok {
				ValidatePluginRegistryConsistency(r, entry, plugin, name)
			}
		}
	}

	fmt.Println(r.Render())
	if r.OK() {
		return 0
	}
	return 1
}
