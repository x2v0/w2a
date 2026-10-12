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

	flag.Parse()

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

	opts := apk.BuildOptions{
		TemplatePath: *template,
		OutputPath:   *out,
		PackageName:  *pkg,
		AppName:      *appName,
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
