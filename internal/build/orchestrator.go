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

func defaultTemplatePath() string {
	// Prefer env override, then our own Java shell.
	if p := os.Getenv("WTA_TEMPLATE"); p != "" {
		return p
	}
	// Look for shells/go-webview/template.apk relative to cwd or binary.
	candidates := []string{
		"shells/go-webview/template.apk",
		filepath.Join(filepath.Dir(os.Args[0]), "..", "shells", "go-webview", "template.apk"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Fallback: legacy path (if user still has app/).
	return "app/src/main/assets/template/webview_shell.apk"
}

var _ = filepath.Base
