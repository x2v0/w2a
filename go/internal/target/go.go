package target

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
)

// GoTarget cross-compiles and injects a Go app as a runtime .so.
type GoTarget struct{}

func (t *GoTarget) Name() string { return "go" }

func (t *GoTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Go.Source)
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

	cfgJSON := map[string]any{
		"appType": "GO",
	}
	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/webtoapp/config.json", data)

	_ = filepath.Base
	return a.SaveAPK(outPath)
}
