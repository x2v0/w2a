package main

import (
	"archive/zip"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
)

func printAssets(zr *zip.Reader) {
	var files []*zip.File
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "assets/") {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		fmt.Println("(no assets)")
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	var total uint64
	fmt.Println("assets/:")
	for _, f := range files {
		rel := strings.TrimPrefix(f.Name, "assets/")
		fmt.Printf("  %-55s %10s\n", rel, humanBytes(f.UncompressedSize64))
		total += f.UncompressedSize64
	}
	fmt.Printf("Total: %d files, %s\n", len(files), humanBytes(total))
}

func printLibs(zr *zip.Reader) {
	abi := map[string][]*zip.File{}
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "lib/") {
			continue
		}
		parts := strings.SplitN(f.Name, "/", 3)
		if len(parts) < 3 {
			continue
		}
		abi[parts[1]] = append(abi[parts[1]], f)
	}
	if len(abi) == 0 {
		fmt.Println("(no native libs)")
		return
	}
	var abis []string
	for a := range abi {
		abis = append(abis, a)
	}
	sort.Strings(abis)
	fmt.Println("lib/:")
	for _, a := range abis {
		var total uint64
		for _, f := range abi[a] {
			total += f.UncompressedSize64
		}
		fmt.Printf("  %s/  (%d files, %s)\n", a, len(abi[a]), humanBytes(total))
		sort.Slice(abi[a], func(i, j int) bool { return abi[a][i].Name < abi[a][j].Name })
		for _, f := range abi[a] {
			base := strings.TrimPrefix(f.Name, "lib/"+a+"/")
			fmt.Printf("    %-50s %10s\n", base, humanBytes(f.UncompressedSize64))
		}
	}
}

func printPermissions(a *apk.APK) {
	perms := a.ListPermissions()
	if len(perms) == 0 {
		fmt.Println("(no permissions declared)")
		return
	}
	sort.Strings(perms)
	fmt.Printf("Permissions (%d):\n", len(perms))
	for _, p := range perms {
		fmt.Printf("  %s\n", p)
	}
}

func printSignature(apkPath string) {
	cmd := exec.Command("apksigner", "verify", "--print-certs", apkPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("(apksigner failed: %v)\n", err)
		return
	}
	lines := strings.Split(string(out), "\n")
	var filtered []string
	for _, l := range lines {
		if strings.HasPrefix(l, "WARNING:") {
			continue
		}
		if strings.HasPrefix(l, "Verified using v") ||
			strings.HasPrefix(l, "Number of signers") ||
			strings.Contains(l, "Signer") && (strings.Contains(l, "certificate") || strings.Contains(l, "digest")) {
			filtered = append(filtered, l)
		}
	}
	if len(filtered) == 0 {
		fmt.Println("(no signature info)")
		return
	}
	fmt.Println("Signature:")
	for _, l := range filtered {
		fmt.Printf("  %s\n", l)
	}
}
