// Command wta-clonehost extracts classes.jar from a clone-host AAR,
// converts it to DEX via d8, and places classes.dex in the assets dir.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/apkutil"
)

func main() {
	var (
		aarPath    = flag.String("aar", "clone-host/build/outputs/aar/clone-host-release.aar", "path to clone-host AAR")
		sdkRoot    = flag.String("sdk", os.Getenv("ANDROID_HOME"), "Android SDK root")
		outDir     = flag.String("out", "app/src/main/assets/clone_host", "output directory for classes.dex")
		workDir    = flag.String("work", "build/clone-host-extract", "intermediate work directory")
		minAPI     = flag.Int("min-api", 23, "minimum API level for d8")
	)
	flag.Parse()

	if *sdkRoot == "" {
		fmt.Fprintln(os.Stderr, "wta-clonehost: -sdk or ANDROID_HOME required")
		os.Exit(1)
	}
	if _, err := os.Stat(*aarPath); err != nil {
		fmt.Fprintf(os.Stderr, "wta-clonehost: AAR not found: %s\n", *aarPath)
		os.Exit(1)
	}

	fmt.Printf("extracting classes.jar from %s\n", *aarPath)
	classesJar, err := apkutil.ExtractClassesJar(*aarPath, *workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract: %v\n", err)
		os.Exit(1)
	}

	buildTools, err := apkutil.FindNewestBuildTool(*sdkRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build-tools: %v\n", err)
		os.Exit(1)
	}
	d8 := filepath.Join(buildTools, "d8"+apkutil.HostBatSuffix())
	if _, err := os.Stat(d8); err != nil {
		d8 = filepath.Join(buildTools, "d8"+apkutil.HostExeSuffix())
	}
	fmt.Printf("using d8: %s\n", d8)

	androidJar, err := apkutil.FindAndroidJar(*sdkRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "android.jar: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("using android.jar: %s\n", androidJar)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir out: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("running d8...")
	if err := apkutil.RunD8(apkutil.D8Options{
		D8Binary:   d8,
		MinAPI:     *minAPI,
		AndroidJar: androidJar,
		OutputDir:  *outDir,
		InputJars:  []string{classesJar},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote classes.dex to %s\n", *outDir)
}
