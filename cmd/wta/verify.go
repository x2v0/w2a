package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
)

type sigInfo struct {
	Scheme string `json:"scheme,omitempty"`
	CN     string `json:"cn,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	SHA1   string `json:"sha1,omitempty"`
	MD5    string `json:"md5,omitempty"`
}

type schemeFlags struct {
	V1 bool `json:"v1"`
	V2 bool `json:"v2"`
	V3 bool `json:"v3"`
	V4 bool `json:"v4"`
}

type verifyResult struct {
	Path        string      `json:"path"`
	Size        int64       `json:"size"`
	Package     string      `json:"package,omitempty"`
	VersionCode uint32      `json:"versionCode,omitempty"`
	VersionName string      `json:"versionName,omitempty"`
	Signature   sigInfo     `json:"signature"`
	Schemes     schemeFlags `json:"schemes"`
	ABIs        []string    `json:"abis,omitempty"`
	AssetsCount int         `json:"assetsCount"`
	AssetsBytes uint64      `json:"assetsBytes"`
	ConfigType  string      `json:"configType,omitempty"`
	PermsCount  int         `json:"permsCount"`
	Errors      []string    `json:"errors,omitempty"`
	Warnings    []string    `json:"warnings,omitempty"`
}

func (r *verifyResult) addErr(msg string)  { r.Errors = append(r.Errors, msg) }
func (r *verifyResult) addWarn(msg string) { r.Warnings = append(r.Warnings, msg) }
func (r *verifyResult) ok() bool           { return len(r.Errors) == 0 }

func cmdVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "machine-readable output")
	strict := fs.Bool("strict", false, "fail if v2 or v3 scheme missing")
	expectPkg := fs.String("expected-pkg", "", "fail if package != value")
	expectABI := fs.String("expected-abi", "", "fail if ABI not present")
	minV1 := fs.Bool("require-v1", false, "fail if v1 missing (Android < 7)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wta verify [flags] <apk>")
		fs.PrintDefaults()
	}
	// reorder: flags first, positional last.
	// Value-taking flags keep their value arg paired.
	{
		valueFlags := map[string]bool{
			"expected-pkg": true,
			"expected-abi": true,
		}
		norm := func(a string) string { return strings.TrimLeft(a, "-") }
		var f, p []string
		for i := 0; i < len(args); i++ {
			a := args[i]
			if strings.HasPrefix(a, "-") {
				f = append(f, a)
				if !strings.Contains(a, "=") && valueFlags[norm(a)] && i+1 < len(args) {
					i++
					f = append(f, args[i])
				}
			} else {
				p = append(p, a)
			}
		}
		args = append(f, p...)
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}
	apkPath := fs.Arg(0)

	r := runVerify(apkPath, *strict, *expectPkg, *expectABI, *minV1)

	if *asJSON {
		out := map[string]any{
			"ok":     r.ok(),
			"result": r,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		if !r.ok() {
			os.Exit(1)
		}
		return
	}

	// text mode
	fmt.Printf("Verifying: %s\n\n", apkPath)
	// signature
	sigStatus := "✓"
	if r.Signature.SHA256 == "" {
		sigStatus = "✗"
	}
	fmt.Printf("  %s signature    %s\n", sigStatus, r.Signature.Scheme)
	if r.Signature.CN != "" {
		// keytool/apksigner already include "CN=" in the value.
		cn := r.Signature.CN
		if !strings.HasPrefix(cn, "CN=") {
			cn = "CN=" + cn
		}
		fmt.Printf("                 %s\n", cn)
	}
	if r.Signature.SHA256 != "" {
		fmt.Printf("                 SHA256=%s\n", r.Signature.SHA256)
	}

	// schemes
	scStatus := "✓"
	if !r.Schemes.V2 && !r.Schemes.V3 {
		scStatus = "⚠"
	}
	var sc []string
	if r.Schemes.V1 {
		sc = append(sc, "v1")
	}
	if r.Schemes.V2 {
		sc = append(sc, "v2")
	}
	if r.Schemes.V3 {
		sc = append(sc, "v3")
	}
	if r.Schemes.V4 {
		sc = append(sc, "v4")
	}
	if len(sc) == 0 {
		sc = []string{"none"}
	}
	fmt.Printf("  %s schemes      %s\n", scStatus, strings.Join(sc, ", "))

	fmt.Printf("  ✓ size         %s\n", humanBytes(uint64(r.Size)))
	fmt.Printf("  ✓ package      %s\n", r.Package)
	fmt.Printf("  ✓ version      %s (%d)\n", r.VersionName, r.VersionCode)
	if len(r.ABIs) > 0 {
		fmt.Printf("  ✓ ABI          %s\n", strings.Join(r.ABIs, ", "))
	} else {
		fmt.Printf("  ⚠ ABI          (no native libs)\n")
	}
	fmt.Printf("  ✓ assets       %d files, %s\n", r.AssetsCount, humanBytes(r.AssetsBytes))
	if r.ConfigType != "" {
		fmt.Printf("  ✓ config       appType: %s\n", r.ConfigType)
	}
	fmt.Printf("  ✓ permissions  %d\n", r.PermsCount)

	// errors / warnings
	if len(r.Errors) > 0 {
		fmt.Println()
		for _, e := range r.Errors {
			fmt.Printf("  ✗ %s\n", e)
		}
	}
	if len(r.Warnings) > 0 {
		fmt.Println()
		for _, w := range r.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}

	fmt.Println()
	if r.ok() {
		fmt.Println("Status: OK")
		return
	}
	fmt.Println("Status: FAIL")
	os.Exit(1)
}

func runVerify(path string, strict bool, expectPkg, expectABI string, requireV1 bool) verifyResult {
	r := verifyResult{Path: path}

	if st, err := os.Stat(path); err == nil {
		r.Size = st.Size()
	} else {
		r.addErr("stat: " + err.Error())
		return r
	}

	// --- apk.LoadAPK: package, version ---
	if a, err := apk.LoadAPK(path); err == nil {
		r.Package = a.Package()
		r.VersionCode = a.VersionCode()
		r.VersionName = a.VersionName()
		r.PermsCount = len(a.ListPermissions())
	} else {
		r.addErr("load apk: " + err.Error())
	}

	// --- apksigner verify ---
	outV, errV := exec.Command("apksigner", "verify", "--verbose", path).CombinedOutput()
	outC, _ := exec.Command("apksigner", "verify", "--print-certs", path).CombinedOutput()

	if errV != nil {
		r.addErr("apksigner verify: " + strings.SplitN(string(outV), "\n", 2)[0])
	}

	// schemes
	for _, line := range strings.Split(string(outV), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Verified using v1 scheme"):
			r.Schemes.V1 = strings.HasSuffix(line, "true")
		case strings.HasPrefix(line, "Verified using v2 scheme"):
			r.Schemes.V2 = strings.HasSuffix(line, "true")
		case strings.HasPrefix(line, "Verified using v3 scheme"):
			r.Schemes.V3 = strings.HasSuffix(line, "true")
		case strings.HasPrefix(line, "Verified using v4 scheme"):
			r.Schemes.V4 = strings.HasSuffix(line, "true")
		}
	}

	// certs
	for _, line := range strings.Split(string(outC), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "certificate DN:") {
			r.Signature.Scheme = strings.SplitN(line, " Signer:", 2)[0]
			r.Signature.CN = strings.TrimSpace(strings.SplitN(line, "certificate DN:", 2)[1])
		} else if strings.Contains(line, "certificate SHA-256 digest:") {
			r.Signature.SHA256 = strings.TrimSpace(strings.SplitN(line, "digest:", 2)[1])
		} else if strings.Contains(line, "certificate SHA-1 digest:") {
			r.Signature.SHA1 = strings.TrimSpace(strings.SplitN(line, "digest:", 2)[1])
		} else if strings.Contains(line, "certificate MD5 digest:") {
			r.Signature.MD5 = strings.TrimSpace(strings.SplitN(line, "digest:", 2)[1])
		}
	}

	// --- zip: ABI, assets, config ---
	if zr, err := zip.OpenReader(path); err == nil {
		abisSet := map[string]bool{}
		for _, f := range zr.File {
			if strings.HasPrefix(f.Name, "lib/") {
				parts := strings.SplitN(f.Name, "/", 3)
				if len(parts) >= 3 {
					abisSet[parts[1]] = true
				}
			}
			if strings.HasPrefix(f.Name, "assets/") {
				r.AssetsCount++
				r.AssetsBytes += f.UncompressedSize64
			}
		}
		var abis []string
		for a := range abisSet {
			abis = append(abis, a)
		}
		sort.Strings(abis)
		r.ABIs = abis

		// app_config.json appType
		for _, f := range zr.File {
			if f.Name == "assets/app_config.json" {
				rc, err := f.Open()
				if err != nil {
					break
				}
				buf := make([]byte, f.UncompressedSize64)
				_, _ = rc.Read(buf)
				_ = rc.Close()
				var cfg map[string]any
				if json.Unmarshal(buf, &cfg) == nil {
					if t, ok := cfg["appType"].(string); ok {
						r.ConfigType = t
					}
				}
				break
			}
		}
		_ = zr.Close()
	}

	// --- checks ---
	if strict {
		if !r.Schemes.V2 {
			r.addErr("strict: v2 scheme missing")
		}
		if !r.Schemes.V3 {
			r.addErr("strict: v3 scheme missing")
		}
	}
	if requireV1 && !r.Schemes.V1 {
		r.addErr("v1 scheme missing (required for Android < 7)")
	}
	if expectPkg != "" && r.Package != expectPkg {
		r.addErr(fmt.Sprintf("package mismatch: got %q, expected %q", r.Package, expectPkg))
	}
	if expectABI != "" {
		found := false
		for _, a := range r.ABIs {
			if a == expectABI {
				found = true
				break
			}
		}
		if !found {
			r.addErr(fmt.Sprintf("expected ABI %q not found (have %v)", expectABI, r.ABIs))
		}
	}

	// sanity warnings (non-fatal)
	if r.Size > 200*1024*1024 {
		r.addWarn(fmt.Sprintf("unusually large APK: %s", humanBytes(uint64(r.Size))))
	}
	if r.Package == "" {
		r.addErr("package name empty (manifest parse failed?)")
	}

	return r
}
