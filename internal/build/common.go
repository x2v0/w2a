package build

import (
	"fmt"
	"os"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/apktool"
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
	noApktool := os.Getenv("WTA_NO_APKTOOL") == "1"

	// apktool path: for manifest edits in-place can't do (longer package,
	// version changes, permissions add/remove, longer label). ~85s overhead.
	apktoolUsed := false
	if !skipManifest && !noApktool && apktool.Available() && needsApktool(cfg) {
		if err := applyManifestViaApktool(cfg, apkPath); err != nil {
			fmt.Printf("apktool: %v (falling back to in-place)\n", err)
		} else {
			fmt.Println("apktool: manifest patched (package/version/perms/label)")
			apktoolUsed = true
			skipManifest = true // version/perms/label already done
		}
	}

	a, err := apk.LoadAPK(apkPath)
	if err != nil {
		return err
	}

	// Application ID in AndroidManifest.xml (in-place path).
	// If apktool used — already correct; SetPackageInPlace is a no-op.
	if !apktoolUsed && cfg.App.Package != "" && cfg.App.Package != "com.webtoapp" {
		if n, err := a.SetPackageInPlace("com.webtoapp", cfg.App.Package); err == nil {
			fmt.Printf("package: replaced %d occurrences → %q\n", n, cfg.App.Package)
		} else {
			fmt.Printf("package: %v (non-fatal)\n", err)
		}
	} else if apktoolUsed {
		fmt.Printf("package: %q (via apktool)\n", cfg.App.Package)
	}

	if !skipManifest {
		if cfg.App.VersionCode > 0 {
			a.SetVersionCode(cfg.App.VersionCode)
		}
		if cfg.App.VersionName != "" {
			a.SetVersionName(cfg.App.VersionName)
		}
	} else if !apktoolUsed {
		fmt.Println("WTA_SKIP_MANIFEST=1 — манифест не изменяется (AXML writer в разработке)")
	}

	// App label — in-place only if not done by apktool.
	if !apktoolUsed && cfg.App.Name != "" && cfg.App.Name != "WebToApp" {
		if n, err := a.SetAppLabelInPlace("WebToApp", cfg.App.Name); err == nil {
			fmt.Printf("label: replaced %d occurrences → %q\n", n, cfg.App.Name)
		} else {
			fmt.Printf("label: %v (non-fatal)\n", err)
		}
	} else if apktoolUsed {
		fmt.Printf("label: %q (via apktool)\n", cfg.App.Name)
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

// needsApktool reports whether the config requires manifest edits beyond
// in-place (which only shortens strings). Returns true for:
//   - package longer than "com.webtoapp"
//   - explicit versionCode / versionName
//   - permissions add/remove
//   - label different from "WebToApp"
func needsApktool(cfg *config.Config) bool {
	if len(cfg.App.Package) > len("com.webtoapp") {
		return true
	}
	if cfg.App.VersionCode > 0 {
		return true
	}
	if cfg.App.VersionName != "" {
		return true
	}
	if len(cfg.Permissions.Add) > 0 || len(cfg.Permissions.Remove) > 0 {
		return true
	}
	if cfg.App.Name != "" && cfg.App.Name != "WebToApp" {
		return true
	}
	return false
}

// applyManifestViaApktool runs apktool d/b on apkPath in-place, applying
// package/version/perms/label from cfg. Falls back to no-op if patch empty.
func applyManifestViaApktool(cfg *config.Config, apkPath string) error {
	data, err := os.ReadFile(apkPath)
	if err != nil {
		return err
	}

	patch := apktool.ManifestPatch{
		Package:     cfg.App.Package,
		AddPerms:    cfg.Permissions.Add,
		RemovePerms: cfg.Permissions.Remove,
		AppLabel:    cfg.App.Name,
	}
	if cfg.App.VersionCode > 0 {
		vc := cfg.App.VersionCode
		patch.VersionCode = &vc
	}
	if cfg.App.VersionName != "" {
		vn := cfg.App.VersionName
		patch.VersionName = &vn
	}

	out, err := apktool.Repack(data, patch)
	if err != nil {
		return err
	}
	return os.WriteFile(apkPath, out, 0o644)
}
