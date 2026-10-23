// Package build orchestrates APK generation from a wta.yaml config.
package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/config"
	"github.com/x2v0/w2a/go/internal/signing"
	"github.com/x2v0/w2a/go/internal/target"
)

// Result holds the outcome of a build.
type Result struct {
	OutputPath string
	Size       int64
	Targets    []string // which target hooks ran
	Signed     bool     // whether the APK was auto-signed
}

// Build orchestrates the pipeline:
//
//	load config → resolve template → apply target → apply common mods →
//	inject stacks → sign → align
func Build(cfg *config.Config) (*Result, error) {
	res := &Result{OutputPath: cfg.Resolve(cfg.Output.Path)}

	// 1. Resolve template.
	templatePath := cfg.Template.APK
	if templatePath == "" {
		templatePath = defaultTemplatePath()
	}
	templatePath = cfg.Resolve(templatePath)
	if _, err := os.Stat(templatePath); err != nil {
		return nil, fmt.Errorf("build: template not found: %s", templatePath)
	}
	// Store resolved path back so downstream (apktool cache key) sees it.
	cfg.Template.APK = templatePath
	fmt.Printf("template: %s\n", templatePath)

	// 2. Load target.
	tgt, err := target.For(cfg.Type)
	if err != nil {
		return nil, fmt.Errorf("build: %w", err)
	}

	// 3. Let target prepare an APK (may be a no-op for WEB/HTML).
	tmpPath := res.OutputPath + ".stage1"
	if err := tgt.Prepare(cfg, templatePath, tmpPath); err != nil {
		return nil, fmt.Errorf("build: %s prepare: %w", cfg.Type, err)
	}
	res.Targets = append(res.Targets, cfg.Type)

	// 4. Apply common modifications.
	if err := applyCommon(cfg, tmpPath); err != nil {
		return nil, fmt.Errorf("build: common mods: %w", err)
	}
	res.Targets = append(res.Targets, "common")

	// 5. Move stage1 to output.
	if err := os.MkdirAll(filepath.Dir(res.OutputPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpPath, res.OutputPath); err != nil {
		return nil, err
	}

	// 5b. Auto-sign (if enabled via sign.Enabled or ~/.config/w2a/sign.yaml).
	if signing.ShouldSign(cfg.Sign) {
		opts := signing.FromConfigAndEnv(cfg.Sign)
		tmpSigned := res.OutputPath + ".signed.tmp"
		if err := signing.Sign(res.OutputPath, tmpSigned, opts); err != nil {
			_ = os.Remove(tmpSigned)
			return nil, fmt.Errorf("build: sign: %w", err)
		}
		if err := os.Rename(tmpSigned, res.OutputPath); err != nil {
			_ = os.Remove(tmpSigned)
			return nil, fmt.Errorf("build: rename signed: %w", err)
		}
		res.Signed = true
		fmt.Printf("sign: signed via %s (alias %s)\n", opts.Keystore, opts.KeyAlias)
	} else {
		fmt.Println("sign: skipped (unsigned)")
	}

	// 6. Size.
	if info, err := os.Stat(res.OutputPath); err == nil {
		res.Size = info.Size()
	}
	return res, nil
}

// defaultTemplatePath returns the shell APK path.
// Priority: wta.yaml `template.apk:` (cfg.Template.APK) > --template flag > shell_webview.apk (auto).
// ENV WTA_TEMPLATE is NOT used (footgun: stale path after rename).
// Per-project override: wta.yaml `template.apk: <path>` or `wta build --template <path>`.
// Global override: ~/.config/w2a/config.yaml (merged into cfg before Build).
func defaultTemplatePath() string {
	exeDir := filepath.Dir(os.Args[0])
	home := os.Getenv("HOME")
	candidates := []string{
		// 1. Relative to cwd (dev from repo root)
		"shells/go-webview/shell_webview.apk",
		// 2. Relative to binary (../shells/...)
		filepath.Join(exeDir, "..", "shells", "go-webview", "shell_webview.apk"),
		// 3. Relative to binary (./shells/...)
		filepath.Join(exeDir, "shells", "go-webview", "shell_webview.apk"),
		// 4. Termux dev default: $HOME/w2a/shells/...
		filepath.Join(home, "w2a", "shells", "go-webview", "shell_webview.apk"),
		// 5. $HOME/shells/...
		filepath.Join(home, "shells", "go-webview", "shell_webview.apk"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Legacy fallback.
	return "app/src/main/assets/template/webview_shell.apk"
}

var _ = filepath.Base
