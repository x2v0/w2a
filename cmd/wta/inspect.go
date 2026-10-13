package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"os"

	"github.com/x2v0/w2a/go/internal/apk"
)

func cmdInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	verbose := fs.Bool("v", false, "verbose (list entries)")
	asJSON := fs.Bool("json", false, "output as JSON")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: wta inspect [flags] <apk>\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}
	path := fs.Arg(0)

	if err := runInspect(path, *verbose, *asJSON); err != nil {
		die(err)
	}
}

func runInspect(path string, verbose, asJSON bool) error {
	a, err := apk.LoadAPK(path)
	if err != nil {
		return err
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer zr.Close()

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
