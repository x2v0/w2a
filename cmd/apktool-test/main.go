package main

import (
	"fmt"
	"os"

	"github.com/x2v0/w2a/go/internal/apktool"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: apktool-test <in.apk> <out.apk> [newpkg]")
		os.Exit(2)
	}
	in, out := os.Args[1], os.Args[2]
	newPkg := "com.example.newlongpackagename"
	if len(os.Args) > 3 {
		newPkg = os.Args[3]
	}

	data, err := os.ReadFile(in)
	if err != nil {
		panic(err)
	}

	patch := apktool.ManifestPatch{
		Package:  newPkg,
		AddPerms: []string{"CAMERA", "RECORD_AUDIO"},
	}

	outData, err := apktool.Repack(data, patch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repack: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, outData, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("[OK] %s → %s (%d bytes)\n", in, out, len(outData))
}
