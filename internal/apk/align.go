package apk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
)

// Align runs zipalign on an APK. pageSize 0 or 4 = default; 16384 = Android 15+.
func Align(srcAPK, dstAPK string, pageSize int) error {
	if pageSize == 0 {
		pageSize = 4
	}
	if pageSize != 4 && pageSize != 16384 {
		return fmt.Errorf("apk: pageSize must be 4 or 16384")
	}
	zipalign, err := findZipalign()
	if err != nil {
		return err
	}
	args := []string{"-f"}
	if pageSize != 4 {
		args = append(args, "-p", fmt.Sprintf("%d", pageSize))
	}
	args = append(args, srcAPK, dstAPK)
	cmd := exec.Command(zipalign, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apk: zipalign failed: %w", err)
	}
	return nil
}

func findZipalign() (string, error) {
	// 1. Explicit override.
	if p := os.Getenv("WTA_ZIPALIGN"); p != "" {
		if isExecutable(p) {
			return p, nil
		}
		return "", fmt.Errorf("apk: WTA_ZIPALIGN=%s not executable on this host", p)
	}

	// 2. Try Termux-native / PATH zipalign first.
	if p, err := exec.LookPath("zipalign"); err == nil && isExecutable(p) {
		return p, nil
	}

	// 3. Scan $ANDROID_HOME/build-tools, newest first, pick first executable.
	sdk := os.Getenv("ANDROID_HOME")
	if sdk == "" {
		sdk = os.Getenv("ANDROID_SDK_ROOT")
	}
	if sdk == "" {
		return "", fmt.Errorf("apk: ANDROID_HOME not set and no zipalign in PATH")
	}
	bt := filepath.Join(sdk, "build-tools")
	entries, err := os.ReadDir(bt)
	if err != nil {
		return "", fmt.Errorf("apk: read build-tools: %w", err)
	}
	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	// Newest first.
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, name := range names {
		p := filepath.Join(bt, name, "zipalign"+exe)
		if _, err := os.Stat(p); err == nil && isExecutable(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("apk: no executable zipalign found under %s (tried %v)", bt, names)
}

// isExecutable returns true if the file exists and can actually run on this
// host (not merely stat-able — catches x86_64 binaries on ARM Termux).
func isExecutable(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	cmd := exec.Command(path, "-h")
	err := cmd.Run()
	if err == nil {
		return true
	}
	// Non-zero exit is fine — the binary ran.
	if _, ok := err.(*exec.ExitError); ok {
		return true
	}
	// "exec format error" / "no such file" / permission denied → not usable.
	return false
}
