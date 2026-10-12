// Command wta-shelltemplate copies the shell-release APK into app assets.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/assetsync"
)

func main() {
	var (
		src = flag.String("src", "shell/build/outputs/apk/release/shell-release.apk", "path to shell APK")
		dst = flag.String("dst", "app/src/main/assets/template/webview_shell.apk", "destination path")
	)
	flag.Parse()

	if _, err := os.Stat(*src); err != nil {
		fmt.Fprintf(os.Stderr, "wta-shelltemplate: APK not found: %s\n", *src)
		os.Exit(1)
	}
	if err := assetsync.CopyFile(*src, *dst); err != nil {
		fmt.Fprintf(os.Stderr, "wta-shelltemplate: %v\n", err)
		os.Exit(1)
	}
	abs, _ := filepath.Abs(*dst)
	fmt.Printf("copied shell template -> %s\n", abs)
}
