package apk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/apkutil"
)

type Toolchain struct {
	SDKRoot    string
	BuildTools string
	AAPT2      string
	D8         string
	APKSigner  string
	Zipalign   string
	AndroidJar string
}

func DiscoverToolchain(sdkRoot string) (*Toolchain, error) {
	if sdkRoot == "" {
		sdkRoot = os.Getenv("ANDROID_HOME")
	}
	if sdkRoot == "" {
		sdkRoot = os.Getenv("ANDROID_SDK_ROOT")
	}
	if sdkRoot == "" {
		return nil, fmt.Errorf("apk: ANDROID_HOME not set")
	}
	bt, err := apkutil.FindNewestBuildTool(sdkRoot)
	if err != nil {
		return nil, err
	}
	jar, err := apkutil.FindAndroidJar(sdkRoot)
	if err != nil {
		return nil, err
	}
	exe := apkutil.HostExeSuffix()
	bat := apkutil.HostBatSuffix()
	return &Toolchain{
		SDKRoot:    sdkRoot,
		BuildTools: bt,
		AAPT2:      filepath.Join(bt, "aapt2"+exe),
		D8:         filepath.Join(bt, "d8"+bat),
		APKSigner:  filepath.Join(bt, "apksigner"+bat),
		Zipalign:   filepath.Join(bt, "zipalign"+exe),
		AndroidJar: jar,
	}, nil
}

func runTool(binary string, args ...string) error {
	cmd := exec.Command(binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apk: %s failed: %w", filepath.Base(binary), err)
	}
	return nil
}

func (tc *Toolchain) AAPT2CompileDir(resDir, outZip string) error {
	return runTool(tc.AAPT2, "compile", "--dir", resDir, "-o", outZip)
}

type APKSignerOptions struct {
	KeystorePath     string
	KeystorePassword string
	KeyAlias         string
	KeyPassword      string
	MinSdkVersion    int
}

func (tc *Toolchain) Sign(apkPath string, o APKSignerOptions) error {
	args := []string{
		"sign",
		"--ks", o.KeystorePath,
		"--ks-pass", "pass:" + o.KeystorePassword,
		"--ks-key-alias", o.KeyAlias,
		"--key-pass", "pass:" + o.KeyPassword,
	}
	if o.MinSdkVersion > 0 {
		args = append(args, "--min-sdk-version", fmt.Sprintf("%d", o.MinSdkVersion))
	}
	args = append(args, apkPath)
	return runTool(tc.APKSigner, args...)
}

func (tc *Toolchain) Verify(apkPath string) error {
	return runTool(tc.APKSigner, "verify", "--verbose", apkPath)
}

func (tc *Toolchain) ZipalignFile(src, dst string, alignment int) error {
	return runTool(tc.Zipalign, "-f", "-p", fmt.Sprintf("%d", alignment), src, dst)
}
