// Command wta-buildapk builds a signed APK from a shell template.
package main

import (
	"flag"
	"fmt"
	"os"

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

	ks := flag.String("keystore", "", "release keystore (skip sign if empty)")
	ksPass := flag.String("ks-pass", "", "keystore password")
	alias := flag.String("key-alias", "", "key alias")
	keyPass := flag.String("key-pass", "", "key password")

	inspect := flag.Bool("inspect", false, "print APK contents and exit")
	verbose := flag.Bool("v", false, "verbose inspect output")
	readPkg := flag.Bool("read-package", false, "print package name and exit")
	listStrings := flag.Bool("list-strings", false, "list strings from resources.arsc and exit")
	listOffset := flag.Int("strings-offset", 0, "start index for -list-strings")
	listLimit := flag.Int("strings-limit", 50, "max strings for -list-strings")
	setAppName := flag.String("set-app-name", "", "replace the app label resource (android:label)")
	setString := flag.String("set-string", "", "format: INDEX=NEW_VALUE — replace a string in the global pool")
	stripArabic := flag.Bool("strip-arabic", false, "replace all Arabic strings in resources.arsc")
	arabicRepl := flag.String("arabic-replacement", "", "replacement value for -strip-arabic")
	normalizeLocale := flag.Bool("normalize-locale", false, "strip locale from ResTableConfig (Arabic→default)")

	flag.Parse()
	_ = setString // TODO: parse INDEX=NEW_VALUE and call SetStringResource

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
