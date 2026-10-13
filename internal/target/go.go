package target

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
)

// GoTarget cross-compiles and injects a Go app as a runtime .so.
type GoTarget struct{}

func (t *GoTarget) Name() string { return "go" }

func (t *GoTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Go.Source)
	// Resolve symlinks (WalkDir does not follow them)
	if resolved, err := filepath.EvalSymlinks(src); err == nil {
		src = resolved
	}
	abis := cfg.Go.ABIs
	if len(abis) == 0 {
		abis = []string{"arm64-v8a", "x86_64"}
	}

	fmt.Printf("go: cross-compiling %s for %v (noCGO=%v)\n", src, abis, cfg.Go.NoCGO)
	soFiles, err := apk.BuildGoRuntime(apk.GoRuntimeOptions{
		SourceDir: src,
		ABIs:      abis,
		NoCGO:     cfg.Go.NoCGO,
		LDFlags:   "-s -w",
	})
	if err != nil {
		return fmt.Errorf("go: build: %w", err)
	}

	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}
	n, err := a.InjectGoRuntime(soFiles)
	if err != nil {
		return err
	}
	fmt.Printf("go: injected %d .so\n", n)

	// Shell (ShellServerLauncher.launchGo) reads from assets/go_app/:
	//   extractAssetsRecursive(context, "go_app", filesDir/go_app_site)
	//   → chmod +x on ELF files → run as process
	//
	// We ALSO place the .so under assets/go_app/ (STORED, so shell can
	// chmod+exec without decompression issues) so shell finds it.
	var binaryName string
	for _, path := range soFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		if err := a.AddAssetStoredBytes("assets/go_app/"+name, data); err != nil {
			return fmt.Errorf("go: add go_app asset: %w", err)
		}
		if binaryName == "" {
			binaryName = name
		}
	}

	targetArch := "arm64-v8a"
	if len(abis) > 0 {
		targetArch = abis[0]
	}

	cfgJSON := map[string]any{
		"appType":  "GO_APP",
		"language": "RUSSIAN",
		"goAppConfig": map[string]any{
			"framework":        "",
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

	return a.SaveAPK(outPath)
}
