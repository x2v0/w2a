package apkutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Aapt2Compile compiles res/ into a flat zip. Returns path to the zip.
func Aapt2Compile(aapt2Binary, resDir, outZip string) error {
	if _, err := os.Stat(aapt2Binary); err != nil {
		return fmt.Errorf("aapt2 not found: %s", aapt2Binary)
	}
	if _, err := os.Stat(resDir); err != nil {
		return fmt.Errorf("res dir not found: %s", resDir)
	}
	cmd := exec.Command(aapt2Binary, "compile", "--dir", resDir, "-o", outZip)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aapt2 compile failed: %v\n%s", err, buf.String())
	}
	return nil
}

// Aapt2Link builds an APK from manifest + optional compiled resources.
func Aapt2Link(aapt2Binary, androidJar, manifest, resZip, outAPK string,
	minSdk, targetSdk int,
) error {
	if _, err := os.Stat(aapt2Binary); err != nil {
		return fmt.Errorf("aapt2 not found: %s", aapt2Binary)
	}
	if _, err := os.Stat(androidJar); err != nil {
		return fmt.Errorf("android.jar not found: %s", androidJar)
	}
	if _, err := os.Stat(manifest); err != nil {
		return fmt.Errorf("manifest not found: %s", manifest)
	}
	args := []string{
		"link",
		"-I", androidJar,
		"--manifest", manifest,
		"--min-sdk-version", fmt.Sprintf("%d", minSdk),
		"--target-sdk-version", fmt.Sprintf("%d", targetSdk),
		"-o", outAPK,
	}
	if resZip != "" {
		if _, err := os.Stat(resZip); err == nil {
			args = append(args, resZip)
		}
	}
	cmd := exec.Command(aapt2Binary, args...)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aapt2 link failed: %v\n%s", err, buf.String())
	}
	return nil
}

// FindAapt2 looks for aapt2 in PATH, then in build-tools.
func FindAapt2(sdkRoot string) (string, error) {
	if p, err := exec.LookPath("aapt2"); err == nil {
		return p, nil
	}
	bt, err := FindNewestBuildTool(sdkRoot)
	if err != nil {
		return "", err
	}
	cand := filepath.Join(bt, "aapt2"+HostExeSuffix())
	if _, err := os.Stat(cand); err != nil {
		return "", fmt.Errorf("aapt2 not in PATH nor in %s", bt)
	}
	return cand, nil
}
