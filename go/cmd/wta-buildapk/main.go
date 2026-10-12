// Command wta-buildapk builds a signed APK from a shell template.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
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

	flag.Parse()
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
}
