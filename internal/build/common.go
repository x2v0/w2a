package build

import (
	"fmt"
	"os"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
	"github.com/x2v0/w2a/go/internal/stack"
)

// applyCommon applies app-wide settings to the staged APK.
//
// Set WTA_SKIP_MANIFEST=1 to disable all AndroidManifest.xml modifications
// (package, version, permissions). Useful while the AXML writer is being
// fixed — aapt2 rejects our output as malformed.
func applyCommon(cfg *config.Config, apkPath string) error {
	skipManifest := os.Getenv("WTA_SKIP_MANIFEST") == "1"

	a, err := apk.LoadAPK(apkPath)
	if err != nil {
		return err
	}

	// Application ID in AndroidManifest.xml (in-place, works even with WTA_SKIP_MANIFEST=1).
	// Shell template package="com.webtoapp" conflicts with host WebToApp (w2a builder).
	// Replaces: package attribute + 3 authorities (fileprovider, firebaseinitprovider, androidx-startup).
	// Classes stay fully-qualified (com.webtoapp.ui.shell.* — do not touch) — match dex.
	if cfg.App.Package != "" && cfg.App.Package != "com.webtoapp" {
		if n, err := a.SetPackageInPlace("com.webtoapp", cfg.App.Package); err == nil {
			fmt.Printf("package: replaced %d occurrences → %q\n", n, cfg.App.Package)
		} else {
			fmt.Printf("package: %v (non-fatal)\n", err)
		}
	}

	if !skipManifest {
		if cfg.App.VersionCode > 0 {
			a.SetVersionCode(cfg.App.VersionCode)
		}
		if cfg.App.VersionName != "" {
			a.SetVersionName(cfg.App.VersionName)
		}
	} else {
		fmt.Println("WTA_SKIP_MANIFEST=1 — манифест не изменяется (AXML writer в разработке)")
	}
	_ = cfg.App.Name

	// App label — using in-place raw patch (see SetAppLabelInPlace).
	if cfg.App.Name != "" && cfg.App.Name != "WebToApp" {
		if n, err := a.SetAppLabelInPlace("WebToApp", cfg.App.Name); err == nil {
			fmt.Printf("label: replaced %d occurrences → %q\n", n, cfg.App.Name)
		} else {
			fmt.Printf("label: %v (non-fatal)\n", err)
		}
	}

	// Icon.
	if cfg.App.Icon != "" {
		iconPath := cfg.Resolve(cfg.App.Icon)
		if _, err := os.Stat(iconPath); err == nil {
			if n, err := a.ReplaceIcon(iconPath); err == nil && n > 0 {
				fmt.Printf("icon: replaced %d mipmaps\n", n)
			}
		}
	}

	// Permissions — also modify AndroidManifest.xml, so skip together.
	if !skipManifest {
		for _, p := range cfg.Permissions.Add {
			if err := a.AddPermission(p); err != nil {
				// Deduplicated — ignore "already present".
			}
		}
		for _, p := range cfg.Permissions.Remove {
			a.RemovePermission(p)
		}
	}

	// Locales.
	// DISABLED: KeepLocales uses arsc.WriteTable → corrupt resources.arsc.
	if false && len(cfg.Locales.Keep) > 0 {
		if n, err := a.KeepLocales(cfg.Locales.Keep); err == nil {
			fmt.Printf("locales: kept %v, stripped %d configs\n", cfg.Locales.Keep, n)
		}
	}
	// DISABLED: StripLocale uses arsc.WriteTable → corrupt resources.arsc.
	for _, tag := range []string{} {
		_ = tag
		if n, err := a.StripLocale(tag); err == nil {
			fmt.Printf("locales: stripped %q (%d configs)\n", tag, n)
		}
	}
	// DISABLED: StripScript uses arsc.WriteTable → corrupt resources.arsc.
	for _, script := range []string{} {
		_ = script
		if n, err := a.StripScript(script, ""); err == nil {
			fmt.Printf("script: stripped %d %s strings\n", n, script)
		}
	}

	// Feature stacks.
	for _, stackPath := range cfg.Features.Stacks {
		p := cfg.Resolve(stackPath)
		b, err := stack.ReadBundle(p)
		if err != nil {
			return fmt.Errorf("stack %s: %w", p, err)
		}
		dex, _ := stack.MergeDex(a, b)
		nl, _ := stack.InjectNativeLibs(a, b)
		na, _ := stack.InjectAssets(a, b, "")
		ps, _ := stack.AppendProguard(a, b)
		nm, _ := stack.MergeManifest(a, b)
		fmt.Printf("stack %s: dex=%v libs=%d assets=%d proguard=%d manifest=%d\n",
			b.Metadata.ID, dex, nl, na, ps, nm)
	}

	// Save.
	return a.SaveAPK(apkPath)
}
