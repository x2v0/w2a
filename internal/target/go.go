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

// GoTarget cross-compiles a Go+web project and packages it as an APK,
// following the w2a shell layout (GoDirectBuilder.kt + ApkBuilder.kt):
//
//  1. go build -buildmode=pie -o bin/<name>   → executable (NOT .so)
//  2. Copy bin/<name>             → assets/go_app/<name>       (STORED)
//  3. Copy source files (main.go, go.mod, static/, templates/)
//     → assets/go_app/...          (shell needs them)
//  4. goAppConfig { binaryName: <name>, targetArch, port, ... }
//  5. libgo_exec_loader.so is already in shell template → shell runs
//     the executable via user-mode execve (W^X bypass).
type GoTarget struct{}

func (t *GoTarget) Name() string { return "go" }

func (t *GoTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Go.Source)
	if resolved, err := filepath.EvalSymlinks(src); err == nil {
		src = resolved
	}
	abis := cfg.Go.ABIs
	if len(abis) == 0 {
		abis = []string{"arm64-v8a"}
	}

	// Binary name from directory (like original: projectDir.name).
	binaryName := filepath.Base(src)

	fmt.Printf("go: cross-compiling %s for %v (noCGO=%v)\n", src, abis, cfg.Go.NoCGO)
	binaries, err := apk.BuildGoRuntime(apk.GoRuntimeOptions{
		SourceDir:  src,
		ABIs:       abis,
		NoCGO:      cfg.Go.NoCGO,
		LDFlags:    "-s -w",
		BuildMode:  "pie", // executable
		BinaryName: binaryName,
	})
	if err != nil {
		return fmt.Errorf("go: build: %w", err)
	}

	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}

	// Copy executable to assets/go_app/<name> (STORED).
	// Shell extracts assets/go_app → filesDir/go_app_site, chmod +x, runs.
	count := 0
	for _, binPath := range binaries {
		data, err := os.ReadFile(binPath)
		if err != nil {
			return fmt.Errorf("go: read binary: %w", err)
		}
		dst := "assets/go_app/" + filepath.Base(binPath)
		if err := a.AddAssetStoredBytes(dst, data); err != nil {
			return fmt.Errorf("go: add binary: %w", err)
		}
		count++
	}

	// Copy source files (main.go, go.mod, static/, templates/, etc.).
	skipDirs := map[string]bool{
		".git": true, "bin": true, "build": true,
		".wta-gobuild": true, ".gradle": true, "node_modules": true,
		// w2a: артефакты сборки не должны попадать в APK —
		// иначе старые бинарники перезапишут свежескомпилированный.
		"dist": true, "_prebuild_bak": true, "_prebuild": true,
		".cache": true, ".idea": true, ".vscode": true,
	}
	walkErr := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "." {
			return nil
		}
		// w2a: не копируем pre-built бинарник с тем же именем, что
		// и скомпилированный — иначе он перезапишет свежий.
		if d.Name() == binaryName {
			return nil
		}
		// w2a: пропускаем бэкапы и мусорные файлы.
		name := d.Name()
		if strings.HasSuffix(name, ".bak") ||
			strings.HasSuffix(name, ".bak2") ||
			strings.HasSuffix(name, ".bak3") ||
			strings.HasSuffix(name, ".old") ||
			strings.HasSuffix(name, "_bak") ||
			strings.HasSuffix(name, "_bak2") ||
			strings.HasSuffix(name, ".manual_bak") ||
			strings.HasSuffix(name, ".orig") ||
			strings.HasSuffix(name, ".swp") ||
			name == ".DS_Store" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return a.AddAssetBytes("assets/go_app/"+rel, data)
	})
	if walkErr != nil {
		return fmt.Errorf("go: copy source: %w", walkErr)
	}

	targetArch := "arm64-v8a"
	if len(abis) > 0 {
		targetArch = abis[0]
	}

	framework := detectGoFramework(src)

	cfgJSON := map[string]any{
		"appType":  "GO_APP",
		"language": "RUSSIAN",
		"goAppConfig": map[string]any{
			"framework":        framework,
			"binaryName":       binaryName,
			"targetArch":       targetArch,
			"port":             0,
			"portConflictMode": "AUTO_KILL",
			"staticDir":        "",
			"envVars":          map[string]string{},
		},
	}
	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/app_config.json", data)

	fmt.Printf("go: injected %d binaries + source tree; binaryName=%s\n", count, binaryName)
	return a.SaveAPK(outPath)
}

func detectGoFramework(srcDir string) string {
	data, err := os.ReadFile(filepath.Join(srcDir, "go.mod"))
	if err != nil {
		return "net/http"
	}
	s := string(data)
	switch {
	case containsAny(s, "gin-gonic/gin"):
		return "gin"
	case containsAny(s, "gofiber/fiber"):
		return "fiber"
	case containsAny(s, "labstack/echo"):
		return "echo"
	case containsAny(s, "gorilla/mux"):
		return "gorilla"
	default:
		return "net/http"
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
