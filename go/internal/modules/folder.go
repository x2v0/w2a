package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// ValidateFolderLayout checks each module folder matches the catalog schema.
func ValidateFolderLayout(r *Report, folder string) {
	name := filepath.Base(folder)
	where := "modules/" + name

	if !KebabCaseRE.MatchString(name) {
		r.Error(where, fmt.Sprintf("folder name must be kebab-case, got %q", name))
	}

	pluginJSON := filepath.Join(folder, "plugin.json")
	pluginHTML := filepath.Join(folder, "plugin.html")
	mainJS := filepath.Join(folder, "main.js")

	if !fileExists(pluginJSON) {
		r.Error(where, "missing required `plugin.json`")
	}
	if !fileExists(pluginHTML) {
		if fileExists(mainJS) {
			r.Warning(where, "legacy multi-file layout (main.js/panel.html) — prefer a single `plugin.html` with a `<script type=\"hcj/page\">` block")
		} else {
			r.Error(where, "missing required `plugin.html`")
		}
	}
	if fileExists(filepath.Join(folder, "module.json")) {
		r.Error(where, "`module.json` is the retired extension-module manifest — remove it; the catalog is plugin.json-only")
	}

	entries, err := os.ReadDir(folder)
	if err != nil {
		return
	}
	for _, e := range entries {
		nm := e.Name()
		if _, ok := ExpectedFolderFiles[nm]; ok {
			continue
		}
		if _, ok := AllowedExtraFiles[nm]; ok {
			continue
		}
		if _, ok := AllowedIconFiles[nm]; ok {
			continue
		}
		if e.IsDir() {
			r.Warning(where, fmt.Sprintf("unexpected sub-directory `%s/` — the runtime won't read it", nm))
		} else {
			r.Warning(where, fmt.Sprintf("unexpected file `%s` — the runtime won't download it", nm))
		}
	}
}

// ValidateIconCoherence cross-checks the contributor icon against registry.
func ValidateIconCoherence(r *Report, folder string, entry map[string]any) {
	name := filepath.Base(folder)
	where := "modules/" + name

	var iconURL string
	if entry != nil {
		if s, ok := entry["iconUrl"].(string); ok {
			iconURL = s
		}
	}

	var present []string
	for f := range AllowedIconFiles {
		if fileExists(filepath.Join(folder, f)) {
			present = append(present, f)
		}
	}
	sort.Strings(present)

	if strings.TrimSpace(iconURL) == "" {
		if len(present) > 0 {
			r.Warning(where, fmt.Sprintf("icon file(s) %v present but registry has no `iconUrl` — they won't be used by the runtime", present))
		}
		return
	}

	iconURL = strings.TrimSpace(iconURL)
	if strings.HasPrefix(iconURL, "http://") || strings.HasPrefix(iconURL, "https://") {
		if strings.HasPrefix(iconURL, "http://") {
			r.Warning(where, "`iconUrl` uses http://; prefer https for catalog assets")
		}
		return
	}

	normalised := strings.TrimLeft(iconURL, "./")
	if strings.HasPrefix(normalised, "/") || strings.Contains(normalised, "..") {
		r.Error(where, fmt.Sprintf("`iconUrl` must be a path inside the module folder, got %q", iconURL))
		return
	}
	if _, ok := AllowedIconFiles[normalised]; !ok {
		r.Error(where, fmt.Sprintf("`iconUrl` must reference one of the allowed icon files, got %q", iconURL))
		return
	}

	iconPath := filepath.Join(folder, normalised)
	info, err := os.Stat(iconPath)
	if err != nil || info.IsDir() {
		r.Error(where, fmt.Sprintf("`iconUrl` points at %q but no such file exists", normalised))
		return
	}
	if info.Size() > MaxIconBytes {
		r.Error(where, fmt.Sprintf("`%s` is %d KB, over the %d KB cap; trim the icon or host it off-repo via an absolute URL",
			normalised, info.Size()/1024, MaxIconBytes/1024))
	}
}

// PageScriptSources returns (where, content) pairs for page-side JS:
// every hcj/page block in plugin.html plus legacy main.js (if present).
func PageScriptSources(r *Report, folder string) [][2]string {
	name := filepath.Base(folder)
	var out [][2]string

	pluginHTML := filepath.Join(folder, "plugin.html")
	if fileExists(pluginHTML) {
		data, err := os.ReadFile(pluginHTML)
		if err != nil {
			return out
		}
		if !utf8.Valid(data) {
			r.Error("modules/"+name+"/plugin.html", "must be UTF-8")
			return out
		}
		html := string(data)
		blocks := HcjPageBlockRE.FindAllStringSubmatch(html, -1)
		if len(blocks) == 0 && !strings.Contains(html, "main.js") {
			r.Warning("modules/"+name+"/plugin.html", "has no `<script type=\"hcj/page\">` block — page-side code will not run")
		}
		if len(blocks) > 0 {
			var parts []string
			for _, b := range blocks {
				parts = append(parts, b[1])
			}
			out = append(out, [2]string{"modules/" + name + "/plugin.html", strings.Join(parts, "\n")})
		}
	}

	mainJS := filepath.Join(folder, "main.js")
	if fileExists(mainJS) {
		data, err := os.ReadFile(mainJS)
		if err == nil {
			if !utf8.Valid(data) {
				r.Error("modules/"+name+"/main.js", "must be UTF-8")
			} else {
				out = append(out, [2]string{"modules/" + name + "/main.js", string(data)})
			}
		}
	}
	return out
}

// ValidateMainJs runs cheap heuristics on page-side scripts.
func ValidateMainJs(r *Report, folder string) {
	sources := PageScriptSources(r, folder)
	for _, s := range sources {
		CheckPageJs(r, folder, s[0], s[1])
	}
}

// CheckPageJs flags empty, oversized, and syntactically suspicious scripts.
func CheckPageJs(r *Report, folder, where, content string) {
	if strings.TrimSpace(content) == "" {
		r.Error(where, "is empty")
		return
	}
	if len(content) > MaxPageScriptBytes {
		r.Warning(where, fmt.Sprintf("is large (%d KB) — consider trimming", len(content)/1024))
	}

	for i, line := range strings.Split(content, "\n") {
		stripped := strings.TrimSpace(line)
		if strings.HasPrefix(stripped, "return ") || stripped == "return;" || stripped == "return" {
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if indent == 0 {
				r.Error(where, fmt.Sprintf("line %d: top-level `return` — the IIFE wrapper would make this a syntax error", i+1))
				break
			}
		}
	}

	if strings.Contains(content, "getConfig(") || strings.Contains(content, "__MODULE_") {
		pluginPath := filepath.Join(folder, "plugin.json")
		if plugin, ok := ReadJSONFile(pluginPath); ok {
			if legacy, _ := plugin["legacyCompat"].(bool); !legacy {
				r.Warning(where, "uses legacy `getConfig`/`__MODULE_*` globals but `plugin.json` does not set `legacyCompat: true`")
			}
		}
	}
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
