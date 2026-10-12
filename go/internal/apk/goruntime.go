package apk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// GoRuntimeOptions configures cross-compilation of a Go app for injection.
type GoRuntimeOptions struct {
	SourceDir string   // directory containing main.go / go.mod
	ABIs      []string // "arm64-v8a", "x86_64" (Android naming)
	PackageID string   // Go package to build (default: ".")
	OutputDir string   // where to place the .so files
	ExtraTags []string // -tags
	LDFlags   string   // -ldflags (e.g. "-s -w")
	BuildMode string   // default "c-shared" — usable as .so
	NoCGO     bool     // CGO_ENABLED=0 — pure-Go static binary (no NDK needed)
}

// GoTargetABI maps Android ABI to GOARCH.
var GoTargetABI = map[string]string{
	"arm64-v8a":   "arm64",
	"x86_64":      "amd64",
	"armeabi-v7a": "arm",
	"x86":         "386",
}

// BuildGoRuntime cross-compiles a Go package for each ABI as a shared
// library (.so) ready for injection into an APK's lib/<abi>/ directory.
// Returns the list of produced .so paths.
func BuildGoRuntime(opts GoRuntimeOptions) ([]string, error) {
	if opts.SourceDir == "" {
		return nil, fmt.Errorf("apk: SourceDir required")
	}
	if len(opts.ABIs) == 0 {
		opts.ABIs = []string{"arm64-v8a", "x86_64"}
	}
	if opts.PackageID == "" {
		opts.PackageID = "."
	}
	if opts.BuildMode == "" {
		opts.BuildMode = "c-shared"
	}
	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Join(opts.SourceDir, "build", "android")
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return nil, err
	}

	// Sanity: go binary.
	if _, err := exec.LookPath("go"); err != nil {
		return nil, fmt.Errorf("apk: 'go' not found in PATH: %w", err)
	}

	var produced []string
	for _, abi := range opts.ABIs {
		goarch, ok := GoTargetABI[abi]
		if !ok {
			return produced, fmt.Errorf("apk: unsupported ABI %q", abi)
		}
		outName := "lib" + sanitizeLibName(opts.PackageID) + ".so"
		// Fallback for "." package: use directory name.
		if opts.PackageID == "." {
			base := filepath.Base(opts.SourceDir)
			outName = "lib" + sanitizeLibName(base) + ".so"
		}
		outPath := filepath.Join(opts.OutputDir, abi, outName)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return produced, err
		}

		buildMode := opts.BuildMode
		var env []string
		if opts.NoCGO {
			env = append(os.Environ(),
				"GOOS=android",
				"GOARCH="+goarch,
				"CGO_ENABLED=0",
			)
			// c-shared needs cgo; use pie for pure-Go.
			if buildMode == "c-shared" {
				buildMode = "pie"
			}
		} else {
			cc, err := ndkCCForABI(abi)
			if err != nil {
				return produced, fmt.Errorf("%w (hint: use -inject-go-no-cgo for pure-Go apps)", err)
			}
			env = append(os.Environ(),
				"GOOS=android",
				"GOARCH="+goarch,
				"CGO_ENABLED=1",
				"CC="+cc,
			)
		}

		args := []string{"build", "-buildmode=" + buildMode}
		if len(opts.ExtraTags) > 0 {
			args = append(args, "-tags", strings.Join(opts.ExtraTags, ","))
		}
		if opts.LDFlags != "" {
			args = append(args, "-ldflags", opts.LDFlags)
		}
		args = append(args, "-o", outPath, opts.PackageID)

		cmd := exec.Command("go", args...)
		cmd.Dir = opts.SourceDir
		cmd.Env = env

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return produced, fmt.Errorf("apk: go build %s: %w", abi, err)
		}
		produced = append(produced, outPath)
	}
	return produced, nil
}

// ndkCCForABI locates the NDK clang for the given Android ABI.
func ndkCCForABI(abi string) (string, error) {
	ndk := os.Getenv("ANDROID_NDK_HOME")
	if ndk == "" {
		ndk = os.Getenv("ANDROID_NDK_ROOT")
	}
	if ndk == "" {
		sdk := os.Getenv("ANDROID_HOME")
		if sdk == "" {
			sdk = os.Getenv("ANDROID_SDK_ROOT")
		}
		if sdk != "" {
			ndkDir := filepath.Join(sdk, "ndk")
			entries, err := os.ReadDir(ndkDir)
			if err == nil {
				var newest string
				for _, e := range entries {
					if e.IsDir() && e.Name() > newest {
						newest = e.Name()
					}
				}
				if newest != "" {
					ndk = filepath.Join(ndkDir, newest)
				}
			}
		}
	}
	if ndk == "" {
		return "", fmt.Errorf("apk: NDK not found (set ANDROID_NDK_HOME)")
	}
	host := hostTag()
	tc := filepath.Join(ndk, "toolchains", "llvm", "prebuilt", host, "bin")
	ccName := map[string]string{
		"arm64-v8a":   "aarch64-linux-android24-clang",
		"x86_64":      "x86_64-linux-android24-clang",
		"armeabi-v7a": "armv7a-linux-androideabi24-clang",
		"x86":         "i686-linux-android24-clang",
	}[abi]
	if ccName == "" {
		return "", fmt.Errorf("apk: no CC mapping for ABI %q", abi)
	}
	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".cmd"
	}
	cc := filepath.Join(tc, ccName+exe)
	if _, err := os.Stat(cc); err != nil {
		return "", fmt.Errorf("apk: clang not found: %s", cc)
	}
	return cc, nil
}

func hostTag() string {
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

func sanitizeLibName(s string) string {
	s = strings.TrimPrefix(s, "./")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	if s == "" || s == "_" {
		return "goapp"
	}
	return s
}

// InjectGoRuntime adds pre-built Go .so files to the APK's lib/<abi>/.
// Each file is stored uncompressed (required for the Android dynamic linker).
// Returns the number of files injected.
func (a *APK) InjectGoRuntime(soFiles []string) (int, error) {
	n := 0
	for _, path := range soFiles {
		// Path format: .../<abi>/<name>.so
		abi := filepath.Base(filepath.Dir(path))
		name := filepath.Base(path)
		if _, ok := GoTargetABI[abi]; !ok {
			continue
		}
		dst := "lib/" + abi + "/" + name
		if err := a.AddNativeLib(path, dst); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
