package apk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type BuildOptions struct {
	TemplatePath string
	OutputPath   string

	PackageName string
	AppName     string
	VersionCode uint32
	VersionName string

	URL        string
	AppType    string
	ConfigJSON []byte
	IconPath   string
	SetAppName string // if non-empty, replace android:label resource

	Keystore APKSignerOptions
	Zipalign bool
}

func Build(opts BuildOptions) error {
	if opts.TemplatePath == "" {
		return fmt.Errorf("apk: template path required")
	}
	if opts.OutputPath == "" {
		return fmt.Errorf("apk: output path required")
	}

	a, err := LoadAPK(opts.TemplatePath)
	if err != nil {
		return err
	}
	fmt.Printf("template: %s\n", opts.TemplatePath)
	fmt.Printf("package:  %s\n", a.Package())

	if opts.PackageName != "" {
		a.SetPackage(opts.PackageName)
		fmt.Printf("-> package: %s\n", opts.PackageName)
	}
	if opts.VersionCode > 0 {
		a.SetVersionCode(opts.VersionCode)
		fmt.Printf("-> versionCode: %d\n", opts.VersionCode)
	}
	if opts.VersionName != "" {
		a.SetVersionName(opts.VersionName)
		fmt.Printf("-> versionName: %s\n", opts.VersionName)
	}

	if opts.SetAppName != "" {
		// Disabled: WriteTable() corrupts large resources.arsc (see aapt2:
		// "corrupt key string pool in ResTable_package"). Enable after the
		// ARSC writer is fixed for large tables.
		_ = opts.SetAppName
	}

	cfg := opts.ConfigJSON
	if cfg == nil {
		cfg = defaultConfig(opts)
	}
	a.Archive.ReplaceDeflated("assets/app_config.json", cfg)
	fmt.Printf("-> config.json: %d bytes\n", len(cfg))

	if opts.IconPath != "" {
		iconBytes, err := os.ReadFile(opts.IconPath)
		if err != nil {
			return fmt.Errorf("apk: read icon: %w", err)
		}
		a.Archive.ReplaceDeflated("assets/webtoapp/icon.png", iconBytes)
		fmt.Printf("-> icon: %d bytes\n", len(iconBytes))
	}

	tmpPath := opts.OutputPath + ".unsigned"
	if err := os.MkdirAll(filepath.Dir(tmpPath), 0o755); err != nil {
		return err
	}
	if err := a.SaveAPK(tmpPath); err != nil {
		return fmt.Errorf("apk: save unsigned: %w", err)
	}
	fmt.Printf("unsigned: %s\n", tmpPath)

	if opts.Keystore.KeystorePath != "" {
		tc, err := DiscoverToolchain("")
		if err != nil {
			return fmt.Errorf("apk: toolchain: %w", err)
		}
		finalTmp := tmpPath
		if opts.Zipalign {
			aligned := opts.OutputPath + ".aligned"
			if err := tc.ZipalignFile(tmpPath, aligned, 4); err != nil {
				return fmt.Errorf("apk: zipalign: %w", err)
			}
			_ = os.Remove(tmpPath)
			finalTmp = aligned
			fmt.Printf("aligned:  %s\n", aligned)
		}
		if err := tc.Sign(finalTmp, opts.Keystore); err != nil {
			return fmt.Errorf("apk: sign: %w", err)
		}
		fmt.Printf("signed:   %s\n", finalTmp)
		if err := os.Rename(finalTmp, opts.OutputPath); err != nil {
			return err
		}
	} else {
		if err := os.Rename(tmpPath, opts.OutputPath); err != nil {
			return err
		}
	}

	fmt.Printf("output:   %s\n", opts.OutputPath)
	return nil
}

func defaultConfig(opts BuildOptions) []byte {
	appType := opts.AppType
	if appType == "" {
		appType = "WEB"
	}
	cfg := map[string]any{
		"appType":   appType,
		"appName":   opts.AppName,
		"targetUrl": opts.URL,
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return data
}
