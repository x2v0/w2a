package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
)

func cmdInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	verbose := fs.Bool("v", false, "verbose (list entries)")
	asJSON := fs.Bool("json", false, "output as JSON")
	showTree := fs.Bool("tree", false, "tree view of APK")
	showSize := fs.Bool("size", false, "size breakdown by type")
	showAssets := fs.Bool("assets", false, "list assets/ entries")
	showLibs := fs.Bool("libs", false, "list lib/ entries per ABI")
	showPerms := fs.Bool("permissions", false, "list declared permissions")
	showSig := fs.Bool("signature", false, "show APK signature (apksigner)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: wta inspect [flags] <apk>\n\n")
		fs.PrintDefaults()
	}
	// Go's flag package stops at first non-flag arg.
	// Reorder: "-flag" args first, positional last, so
	//   wta inspect <apk> --tree
	// parses --tree correctly.
	{
		var flagArgs, posArgs []string
		for _, a := range args {
			if strings.HasPrefix(a, "-") {
				flagArgs = append(flagArgs, a)
			} else {
				posArgs = append(posArgs, a)
			}
		}
		args = append(flagArgs, posArgs...)
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}
	path := fs.Arg(0)

	if err := runInspect(path, *verbose, *asJSON, *showTree, *showSize, *showAssets, *showLibs, *showPerms, *showSig); err != nil {
		die(err)
	}
}

func runInspect(path string, verbose, asJSON, showTree, showSize, showAssets, showLibs, showPerms, showSig bool) error {
	a, err := apk.LoadAPK(path)
	if err != nil {
		return err
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer zr.Close()

	// tree / size modes (early return, text-only)
	if showTree || showSize {
		var totalSize int64
		if st, err := os.Stat(path); err == nil {
			totalSize = st.Size()
		}
		if showTree {
			fmt.Printf("%s (%s compressed)\n", path, humanBytes(uint64(totalSize)))
			renderTree(buildTree(zr.File), "")
			if showSize {
				fmt.Println()
			}
		}
		if showSize {
			printSize(&zr.Reader)
		}
		return nil
	}

	// detail views (early return, text-only)
	if showAssets || showLibs || showPerms || showSig {
		first := true
		sep := func() {
			if !first {
				fmt.Println()
			}
			first = false
		}
		if showAssets {
			sep()
			printAssets(&zr.Reader)
		}
		if showLibs {
			sep()
			printLibs(&zr.Reader)
		}
		if showPerms {
			sep()
			printPermissions(a)
		}
		if showSig {
			sep()
			printSignature(path)
		}
		return nil
	}

	var stored, deflated int
	for _, f := range zr.File {
		switch f.Method {
		case zip.Store:
			stored++
		case zip.Deflate:
			deflated++
		}
	}

	// Config contents if present
	cfgText := ""
	for _, name := range []string{"assets/app_config.json", "assets/webtoapp/config.json"} {
		for _, f := range zr.File {
			if f.Name == name {
				rc, err := f.Open()
				if err == nil {
					buf := make([]byte, f.UncompressedSize64)
					rc.Read(buf)
					rc.Close()
					cfgText = string(buf)
				}
				break
			}
		}
		if cfgText != "" {
			break
		}
	}

	if asJSON {
		fmt.Printf(`{"file":%q,"package":%q,"versionCode":%d,"versionName":%q,`+
			`"entries":%d,"stored":%d,"deflated":%d,"config":%q}`+"\n",
			path, a.Package(), a.VersionCode(), a.VersionName(),
			len(a.Archive.Entries), stored, deflated, cfgText)
		return nil
	}

	fmt.Printf("file:        %s\n", path)
	fmt.Printf("package:     %s\n", a.Package())
	fmt.Printf("versionCode: %d\n", a.VersionCode())
	fmt.Printf("versionName: %s\n", a.VersionName())
	fmt.Printf("entries:     %d\n", len(a.Archive.Entries))
	fmt.Printf("stored:      %d\n", stored)
	fmt.Printf("deflated:    %d\n", deflated)

	if cfgText != "" {
		fmt.Printf("--- config ---\n%s\n", cfgText)
	}

	if verbose {
		fmt.Println("--- entries ---")
		for _, f := range zr.File {
			m := "Deflate"
			if f.Method == zip.Store {
				m = "Stored"
			}
			fmt.Printf("  %-50s %10d  %s\n", f.Name, f.UncompressedSize64, m)
		}
	}
	return nil
}
