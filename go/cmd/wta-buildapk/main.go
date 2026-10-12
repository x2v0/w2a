// Command wta-buildapk builds a signed APK from a shell template.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/cache"
	"github.com/x2v0/w2a/go/internal/stack"
)

func main() {
	template := flag.String("template", "app/src/main/assets/template/webview_shell.apk", "shell template APK")
	out := flag.String("out", "build/output.apk", "output APK path")
	pkg := flag.String("package", "", "package name")
	appName := flag.String("app-name", "", "app name")
	url := flag.String("url", "", "target URL")
	appType := flag.String("app-type", "WEB", "app type")
	versionCode := flag.Uint("version-code", 0, "versionCode (0 = keep)")
	versionName := flag.String("version-name", "", "versionName (keep if empty)")
	iconPath := flag.String("icon", "", "PNG icon path")
	zipalign := flag.Bool("zipalign", true, "run zipalign")

	ks := flag.String("keystore", "", "release keystore")
	ksPass := flag.String("ks-pass", "", "keystore password")
	alias := flag.String("key-alias", "", "key alias")
	keyPass := flag.String("key-pass", "", "key password")

	inspect := flag.Bool("inspect", false, "print APK contents and exit")
	verbose := flag.Bool("v", false, "verbose inspect output")
	readPkg := flag.Bool("read-package", false, "print package name and exit")
	listStrings := flag.Bool("list-strings", false, "list strings and exit")
	listOffset := flag.Int("strings-offset", 0, "start index")
	listLimit := flag.Int("strings-limit", 50, "max strings")
	listLocales := flag.Bool("list-locales", false, "list locales and exit")
	stripLocale := flag.String("strip-locale", "", "strip one locale")
	stripLocales := flag.String("strip-locales", "", "strip multiple locales CSV")
	keepLocales := flag.String("keep-locales", "", "keep only these locales CSV")
	stripScript := flag.String("strip-script", "", "strip strings by script")
	scriptRepl := flag.String("script-replacement", "", "replacement for -strip-script")
	stripArabic := flag.Bool("strip-arabic", false, "strip Arabic")
	arabicRepl := flag.String("arabic-replacement", "", "replacement for -strip-arabic")
	normalizeLocale := flag.Bool("normalize-locale", false, "zero all locales")
	replaceIcon := flag.String("replace-icon", "", "replace ic_launcher PNGs")
	setAppName := flag.String("set-app-name", "", "replace android:label")
	setString := flag.String("set-string", "", "INDEX=NEW_VALUE")
	addPermission := flag.String("add-permission", "", "add <uses-permission> (comma-separated)")
	removePermission := flag.String("remove-permission", "", "remove permission (comma-separated)")
	listPermissions := flag.Bool("list-permissions", false, "list <uses-permission> and exit")
	addAsset := flag.String("add-asset", "", "add asset: src=assets/dst")
	listAssets := flag.Bool("list-assets", false, "list assets/* entries")
	addSO := flag.String("add-so", "", "add native lib: src.so=lib/abi/name.so")
	alignPages := flag.Int("align-pages", 0, "zipalign to N-byte pages (4=default, 16384=Android 15+)")
	useCache := flag.Bool("cache", false, "skip build if inputs unchanged")
	clearCache := flag.Bool("clear-cache", false, "wipe cache and exit")
	injectGo := flag.String("inject-go", "", "path to Go source dir; cross-compile and inject .so")
	injectGoABIs := flag.String("inject-go-abis", "arm64-v8a,x86_64", "ABIs for -inject-go")
	injectGoNoCGO := flag.Bool("inject-go-no-cgo", false, "use CGO_ENABLED=0 (pure Go, no NDK)")
	injectStack := flag.String("inject-stack", "", "path to feature-stack .zip (admob.zip etc.)")
	injectStackNS := flag.String("inject-stack-namespace", "", "optional asset subdir namespace")

	flag.Parse()

	if *clearCache {
		s, err := cache.Open("")
		if err == nil {
			s.Clear()
			_ = s.Save()
			fmt.Println("cache cleared")
		}
		return
	}
	_ = setString
	_ = setAppName

	if *inspect {
		if err := runInspect(*template, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *readPkg {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(a.Package())
		return
	}
	if *listStrings {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		strs, err := a.ListStrings(*listOffset, *listLimit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		for i, s := range strs {
			fmt.Printf("[%d] %s\n", *listOffset+i, s)
		}
		return
	}
	if *listLocales {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		locs, err := a.ListLocales()
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		for _, l := range locs {
			fmt.Printf("%-12s %d type(s)\n", l.Tag, l.TypeCount)
		}
		return
	}
	if *stripLocale != "" || *stripLocales != "" || *keepLocales != "" {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		var total int
		if *stripLocale != "" {
			n, err := a.StripLocale(*stripLocale)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
				os.Exit(1)
			}
			total += n
			fmt.Printf("stripped locale %q: %d configs\n", *stripLocale, n)
		}
		if *stripLocales != "" {
			for _, tag := range strings.Split(*stripLocales, ",") {
				tag = strings.TrimSpace(tag)
				if tag == "" {
					continue
				}
				n, err := a.StripLocale(tag)
				if err != nil {
					fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
					os.Exit(1)
				}
				total += n
				fmt.Printf("stripped locale %q: %d configs\n", tag, n)
			}
		}
		if *keepLocales != "" {
			keep := []string{}
			for _, tag := range strings.Split(*keepLocales, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					keep = append(keep, tag)
				}
			}
			n, err := a.KeepLocales(keep)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
				os.Exit(1)
			}
			total += n
			fmt.Printf("kept %v, stripped %d other configs\n", keep, n)
		}
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("total %d configs modified\noutput:   %s\n", total, *out)
		return
	}

	if *stripScript != "" {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		n, err := a.StripScript(*stripScript, *scriptRepl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("stripped %d %s strings\n", n, *stripScript)
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("output:   %s\n", *out)
		return
	}
	if *stripArabic {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		n, err := a.StripArabic(*arabicRepl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("stripped %d Arabic strings\n", n)
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("output:   %s\n", *out)
		return
	}
	if *normalizeLocale {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		n, err := a.NormalizeLocale()
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("normalized %d locale configs\n", n)
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("output:   %s\n", *out)
		return
	}
	if *replaceIcon != "" {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		n, err := a.ReplaceIcon(*replaceIcon)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("replaced %d icon entries\n", n)
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("output:   %s\n", *out)
		return
	}

	if *listPermissions {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		for _, p := range a.ListPermissions() {
			fmt.Println(p)
		}
		return
	}

	if *addPermission != "" || *removePermission != "" {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		if *addPermission != "" {
			for _, name := range strings.Split(*addPermission, ",") {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				if err := a.AddPermission(name); err != nil {
					fmt.Printf("skip: %v\n", err)
				} else {
					fmt.Printf("added permission: %s\n", name)
				}
			}
		}
		if *removePermission != "" {
			for _, name := range strings.Split(*removePermission, ",") {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				n := a.RemovePermission(name)
				fmt.Printf("removed %d entries: %s\n", n, name)
			}
		}
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("output:   %s\n", *out)
		return
	}

	if *listAssets {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		for _, name := range a.ListAssets() {
			fmt.Println(name)
		}
		return
	}

	if *addAsset != "" {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		parts := strings.SplitN(*addAsset, "=", 2)
		if len(parts) != 2 {
			fmt.Fprintln(os.Stderr, "wta-buildapk: -add-asset format: src=assets/dst")
			os.Exit(1)
		}
		if err := a.AddAsset(parts[0], parts[1]); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("asset added: %s\n", parts[1])
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("output:   %s\n", *out)
		return
	}

	if *addSO != "" {
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		parts := strings.SplitN(*addSO, "=", 2)
		if len(parts) != 2 {
			fmt.Fprintln(os.Stderr, "wta-buildapk: -add-so format: src.so=lib/abi/name.so")
			os.Exit(1)
		}
		if err := a.AddNativeLib(parts[0], parts[1]); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("native lib added: %s\n", parts[1])
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("output:   %s\n", *out)
		return
	}

	if *alignPages > 0 {
		if err := apk.Align(*template, *out, *alignPages); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("aligned: %s (%d-byte pages)\n", *out, *alignPages)
		return
	}

	if *useCache {
		store, err := cache.Open("")
		if err == nil {
			inputs := []string{*template}
			if *iconPath != "" {
				inputs = append(inputs, *iconPath)
			}
			fp, fpErr := cache.HashFiles(inputs)
			if fpErr == nil {
				if e, ok := store.Get(*out, fp); ok {
					fmt.Printf("cache hit: %s (built %s)\n", e.Output, e.BuiltAt.Format("2006-01-02 15:04"))
					return
				}
				fmt.Println("cache miss — rebuilding")
			}
		}
	}

	if *injectGo != "" {
		abis := []string{}
		for _, a := range strings.Split(*injectGoABIs, ",") {
			a = strings.TrimSpace(a)
			if a != "" {
				abis = append(abis, a)
			}
		}
		fmt.Printf("cross-compiling Go app: %s -> %v\n", *injectGo, abis)
		soFiles, err := apk.BuildGoRuntime(apk.GoRuntimeOptions{
			NoCGO:     *injectGoNoCGO,
			SourceDir: *injectGo,
			ABIs:      abis,
			LDFlags:   "-s -w",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		for _, so := range soFiles {
			fmt.Printf("  built: %s\n", so)
		}
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		n, err := a.InjectGoRuntime(soFiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("injected %d .so file(s)\n", n)
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("output:   %s\n", *out)
		return
	}

	if *injectStack != "" {
		b, err := stack.ReadBundle(*injectStack)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("stack: %s v%d (pkg-id %s)\n", b.Metadata.ID, b.Metadata.Version, b.Metadata.PackageID)
		a, err := apk.LoadAPK(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		dex, err := stack.MergeDex(a, b)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  dex injected: %v\n", dex)
		na, err := stack.InjectAssets(a, b, *injectStackNS)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  assets injected: %d\n", na)
		nm, err := stack.MergeManifest(a, b)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  manifest nodes merged: %d\n", nm)
		if err := a.SaveAPK(*out); err != nil {
			fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("output:   %s\n", *out)
		return
	}

	effectiveName := *appName
	if *setAppName != "" {
		effectiveName = *setAppName
	}
	opts := apk.BuildOptions{
		TemplatePath: *template,
		OutputPath:   *out,
		PackageName:  *pkg,
		AppName:      effectiveName,
		SetAppName:   *setAppName,
		VersionCode:  uint32(*versionCode),
		VersionName:  *versionName,
		URL:          *url,
		AppType:      *appType,
		IconPath:     *iconPath,
		Zipalign:     *zipalign,
	}
	if *ks != "" {
		opts.Keystore = apk.APKSignerOptions{
			KeystorePath:     *ks,
			KeystorePassword: *ksPass,
			KeyAlias:         *alias,
			KeyPassword:      *keyPass,
			MinSdkVersion:    23,
		}
	}
	if err := apk.Build(opts); err != nil {
		fmt.Fprintf(os.Stderr, "wta-buildapk: %v\n", err)
		os.Exit(1)
	}

	if *useCache {
		store, err := cache.Open("")
		if err == nil {
			inputs := []string{*template}
			if *iconPath != "" {
				inputs = append(inputs, *iconPath)
			}
			if fp, err := cache.HashFiles(inputs); err == nil {
				outFP := cache.Empty
				if outData, err := os.ReadFile(*out); err == nil {
					outFP = cache.HashBytes(outData)
				}
				store.Put(*out, cache.Entry{
					Fingerprint: fp,
					Output:      *out,
					OutputHash:  outFP,
					BuiltAt:     time.Now(),
				})
				_ = store.Save()
			}
		}
	}
}
