// Package apkutil contains helpers for manipulating AAR/APK/DEX artifacts.
package apkutil

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractClassesJar extracts classes.jar from an AAR (or APK) and writes it
// to outDir/classes.jar. Returns the path to the extracted jar.
func ExtractClassesJar(aarPath, outDir string) (string, error) {
	zr, err := zip.OpenReader(aarPath)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer zr.Close()

	var entry *zip.File
	for _, f := range zr.File {
		if f.Name == "classes.jar" {
			entry = f
			break
		}
	}
	if entry == nil {
		return "", fmt.Errorf("classes.jar not found in %s", aarPath)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	dst := filepath.Join(outDir, "classes.jar")
	src, err := entry.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}
	return dst, nil
}

// FindNewestBuildTool finds the newest numbered subdirectory inside
// sdkRoot/build-tools/ (e.g. "36.1.0"). Returns its absolute path.
func FindNewestBuildTool(sdkRoot string) (string, error) {
	base := filepath.Join(sdkRoot, "build-tools")
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("read build-tools: %w", err)
	}
	var best string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Compare with simple lexicographic-on-padded-numerics.
		if best == "" || compareVersion(name, best) > 0 {
			best = name
		}
	}
	if best == "" {
		return "", fmt.Errorf("no build-tools under %s", base)
	}
	return filepath.Join(base, best), nil
}

// compareVersion compares dotted version strings numerically.
func compareVersion(a, b string) int {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var na, nb int
		if i < len(pa) {
			na = atoiSafe(pa[i])
		}
		if i < len(pb) {
			nb = atoiSafe(pb[i])
		}
		if na != nb {
			if na > nb {
				return 1
			}
			return -1
		}
	}
	return 0
}

func atoiSafe(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

// HostExeSuffix returns ".exe" on Windows, "" elsewhere.
func HostExeSuffix() string {
	if isWindows() {
		return ".exe"
	}
	return ""
}

// HostBatSuffix returns ".bat" on Windows, "" elsewhere.
func HostBatSuffix() string {
	if isWindows() {
		return ".bat"
	}
	return ""
}
