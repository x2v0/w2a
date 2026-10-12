package musl

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func HostTag() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "darwin-arm64"
		}
		return "darwin-x86_64"
	case "windows":
		return "windows-x86_64"
	default:
		return "linux-x86_64"
	}
}

func DetectNDK() (string, error) {
	for _, k := range []string{"ANDROID_NDK_HOME", "ANDROID_NDK_ROOT", "NDK_ROOT"} {
		v := os.Getenv(k)
		if v == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(v, "toolchains")); err == nil {
			return v, nil
		}
	}
	for _, k := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		sdk := os.Getenv(k)
		if sdk == "" {
			continue
		}
		ndkDir := filepath.Join(sdk, "ndk")
		entries, err := os.ReadDir(ndkDir)
		if err != nil {
			continue
		}
		var latest string
		for _, e := range entries {
			if e.IsDir() && e.Name() > latest {
				latest = e.Name()
			}
		}
		if latest != "" {
			return filepath.Join(ndkDir, latest), nil
		}
	}
	return "", fmt.Errorf("NDK not found (set ANDROID_NDK_HOME or ANDROID_SDK_ROOT)")
}

func toolchainDir(ndk string) (string, error) {
	if ndk == "" {
		var err error
		ndk, err = DetectNDK()
		if err != nil {
			return "", err
		}
	}
	tc := filepath.Join(ndk, "toolchains", "llvm", "prebuilt", HostTag())
	if _, err := os.Stat(tc); err != nil {
		return "", fmt.Errorf("toolchain not found at %s", tc)
	}
	return tc, nil
}

func withExe(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}
	if _, err := os.Stat(path + ".exe"); err == nil {
		return path + ".exe"
	}
	return path
}
