package main

import (
	"archive/zip"
	"fmt"

	"github.com/x2v0/w2a/go/internal/apk"
)

func runInspect(path string, verbose bool) error {
	a, err := apk.LoadAPK(path)
	if err != nil {
		return err
	}
	fmt.Printf("file:        %s\n", path)
	fmt.Printf("package:     %s\n", a.Package())
	fmt.Printf("versionCode: %d\n", a.VersionCode())
	fmt.Printf("versionName: %s\n", a.VersionName())
	fmt.Printf("entries:     %d\n", len(a.Archive.Entries))

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
	fmt.Printf("stored:      %d\n", stored)
	fmt.Printf("deflated:    %d\n", deflated)

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
