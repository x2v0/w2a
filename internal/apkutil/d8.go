package apkutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// D8Options holds the parameters needed to invoke d8.
type D8Options struct {
	D8Binary  string // absolute path to d8 / d8.bat
	MinAPI    int    // e.g. 23
	AndroidJar string // absolute path to android.jar
	OutputDir string // where dex files are written
	InputJars []string // input jar files
}

// RunD8 invokes the Android D8 tool and returns any captured stderr.
func RunD8(opts D8Options) error {
	if _, err := os.Stat(opts.D8Binary); err != nil {
		return fmt.Errorf("d8 binary not found: %s", opts.D8Binary)
	}
	if _, err := os.Stat(opts.AndroidJar); err != nil {
		return fmt.Errorf("android.jar not found: %s", opts.AndroidJar)
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return err
	}

	args := []string{
		"--release",
		"--min-api", fmt.Sprintf("%d", opts.MinAPI),
		"--lib", opts.AndroidJar,
		"--output", opts.OutputDir,
	}
	args = append(args, opts.InputJars...)

	cmd := exec.Command(opts.D8Binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("d8 failed: %v\n%s", err, stderr.String())
	}
	return nil
}

// FindAndroidJar returns the path to android.jar for the highest installed
// platform under sdkRoot/platforms/.
func FindAndroidJar(sdkRoot string) (string, error) {
	base := filepath.Join(sdkRoot, "platforms")
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("read platforms: %w", err)
	}
	var best string
	for _, e := range entries {
		if !e.IsDir() || !hasPrefix(e.Name(), "android-") {
			continue
		}
		ver := e.Name()[len("android-"):]
		if best == "" || atoiSafe(ver) > atoiSafe(best) {
			best = ver
		}
	}
	if best == "" {
		return "", fmt.Errorf("no android-* under %s", base)
	}
	jar := filepath.Join(base, "android-"+best, "android.jar")
	if _, err := os.Stat(jar); err != nil {
		return "", fmt.Errorf("android.jar missing: %s", jar)
	}
	return jar, nil
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
