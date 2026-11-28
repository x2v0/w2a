package build

import (
	"fmt"
	"os"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/apktool"
	"github.com/x2v0/w2a/go/internal/config"
)

// applyCommon applies app-wide settings to the staged APK.
//
// Set WTA_SKIP_MANIFEST=1 to disable all AndroidManifest.xml modifications
// (package, version, permissions). Useful while the AXML writer is being
// fixed — aapt2 rejects our output as malformed.
func applyCommon(cfg *config.Config, apkPath string) error {
	skipManifest := os.Getenv("WTA_SKIP_MANIFEST") == "1"
	noApktool := os.Getenv("WTA_NO_APKTOOL") == "1"

	// Load template FIRST: needsApktool compares against actual template
	// values (avoid apktool when package/version/label/permissions
	// match template exactly — in-place is safe then).
	a, err := apk.LoadAPK(apkPath)
	if err != nil {
		return err
	}

	// apktool path: for manifest edits in-place can't do (longer package,
	// version changes, permissions add/remove, longer label). ~85s overhead.
	apktoolUsed := false
	if !skipManifest && !noApktool && apktool.Available() && needsApktool(cfg, a) {
		if err := applyManifestViaApktool(cfg, apkPath); err != nil {
			fmt.Printf("apktool: %v (falling back to in-place)\n", err)
		} else {
			fmt.Println("apktool: manifest patched (package/version/perms/label)")
			apktoolUsed = true
			skipManifest = true // version/perms/label already done
			// Reload patched APK
			a, err = apk.LoadAPK(apkPath)
			if err != nil {
				return err
			}
		}
	}

	// Application ID in AndroidManifest.xml (in-place path).
	// If apktool used — already correct; SetPackageInPlace is a no-op.
	if !skipManifest && !apktoolUsed && cfg.App.Package != "" && cfg.App.Package != "com.webtoapp" {
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
	// Try known template labels in order (our shell = "WTA App",
	// old WebToApp shell = "WebToApp"). SetAppLabelInPlace only shortens.
	if !skipManifest && !apktoolUsed && cfg.App.Name != "" {
		replaced := false
		for _, oldLabel := range []string{"WTA App", "WebToApp"} {
			if cfg.App.Name == oldLabel {
				replaced = true
				break
			}
			if len(cfg.App.Name) > len(oldLabel) {
				continue
			}
			if n, err := a.SetAppLabelInPlace(oldLabel, cfg.App.Name); err == nil && n > 0 {
				fmt.Printf("label: replaced %d occurrences of %q → %q\n", n, oldLabel, cfg.App.Name)
				replaced = true
				break
			}
		}
		if !replaced && cfg.App.Name != "WebToApp" {
			fmt.Printf("label: no matching base label (tried WTA App, WebToApp) — skipped\n")
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

	// Save.
	return a.SaveAPK(apkPath)
}

// needsApktool reports whether the config requires manifest edits that
// in-place can't do safely. Compares against the ACTUAL template (a):
//   - package longer than template's package
//   - versionCode/Name differ from template
//   - permissions to add that template doesn't already have
//   - permissions to remove that template actually has
//   - label longer than template's label
//
// If everything matches the template — in-place is safe (3s vs 100s).
func needsApktool(cfg *config.Config, a *apk.APK) bool {
	tmplPkg := a.Package()
	tmplVC := a.VersionCode()
	tmplVN := a.VersionName()

	// package: apktool only if new is longer (in-place shortens with padding)
	if cfg.App.Package != "" && cfg.App.Package != tmplPkg &&
		len(cfg.App.Package) > len(tmplPkg) {
		return true
	}

	// versionCode: differs from template
	if cfg.App.VersionCode > 0 && cfg.App.VersionCode != tmplVC {
		return true
	}

	// versionName: differs from template (string length matters for in-place)
	if cfg.App.VersionName != "" && cfg.App.VersionName != tmplVN {
		return true
	}

	// permissions: add only if template lacks it; remove only if template has it
	if len(cfg.Permissions.Add) > 0 || len(cfg.Permissions.Remove) > 0 {
		have := map[string]bool{}
		for _, p := range a.ListPermissions() {
			have[p] = true
		}
		for _, p := range cfg.Permissions.Add {
			if !have[p] {
				return true
			}
		}
		for _, p := range cfg.Permissions.Remove {
			if have[p] {
				return true
			}
		}
	}

	// label: apktool only if longer than template's label.
	// Our shell = "WTA App" (7), old WebToApp shell = "WebToApp" (8).
	// Use min (7) — conservative (slightly more apktool for 8-char names).
	if cfg.App.Name != "" && cfg.App.Name != "WebToApp" &&
		len(cfg.App.Name) > len("WTA App") {
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

	// Cache key = ORIGINAL template bytes (stable across builds).
	// apkPath includes injected assets + volatile zip timestamps → bad key.
	var cacheKey []byte
	if cfg.Template.APK != "" {
		if b, err := os.ReadFile(cfg.Resolve(cfg.Template.APK)); err == nil {
			cacheKey = b
		}
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

	out, err := apktool.Repack(data, patch, cacheKey)
	if err != nil {
		return err
	}
	return os.WriteFile(apkPath, out, 0o644)
}
