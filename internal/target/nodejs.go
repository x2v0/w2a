package target

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
)

// NodeJSTarget packages a Node.js project as a NODEJS_APP APK.
type NodeJSTarget struct{}

func (t *NodeJSTarget) Name() string { return "nodejs" }

const (
	nodeVersionZip = "18.20.4-capawesome.1"
	nodeZipURL     = "https://github.com/capawesome-team/nodejs-mobile/releases/download/v18.20.4-capawesome.1/nodejs-mobile-v18.20.4-capawesome.1-android.zip"
	nodeZipSHA256  = "1b3c7979c81aec89a7f51b29af1f4875a5d637727ad6e2c392cdf2e127715da9"
)

var nodeZipPaths = map[string]string{
	"arm64-v8a":   "bin/arm64-v8a/libnode.so",
	"armeabi-v7a": "bin/armeabi-v7a/libnode.so",
	"x86_64":      "bin/x86_64/libnode.so",
}

var ndkLibcxxPaths = map[string]string{
	"arm64-v8a":   "toolchains/llvm/prebuilt/<host>/sysroot/usr/lib/aarch64-linux-android/libc++_shared.so",
	"armeabi-v7a": "toolchains/llvm/prebuilt/<host>/sysroot/usr/lib/arm-linux-androideabi/libc++_shared.so",
	"x86_64":      "toolchains/llvm/prebuilt/<host>/sysroot/usr/lib/x86_64-linux-android/libc++_shared.so",
}

func (t *NodeJSTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	if cfg.NodeJS == nil {
		return fmt.Errorf("nodejs: cfg.NodeJS is nil")
	}

	src := cfg.Resolve(cfg.NodeJS.Source)
	if resolved, err := filepath.EvalSymlinks(src); err == nil {
		src = resolved
	}

	abis := cfg.NodeJS.ABIs
	if len(abis) == 0 {
		abis = []string{"arm64-v8a"}
	}

	entry := cfg.NodeJS.EntryModule
	if entry == "" {
		entry = "index.js"
	}

	fmt.Printf("nodejs: packaging %s for %v (entry=%s)\n", src, abis, entry)

	libnodePaths, err := ensureLibnodeCached(abis)
	if err != nil {
		return fmt.Errorf("nodejs: libnode: %w", err)
	}
	libcxxPaths, err := resolveLibcxxPaths(abis)
	if err != nil {
		return fmt.Errorf("nodejs: libc++_shared: %w", err)
	}

	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}

	for _, abi := range abis {
		p := libnodePaths[abi]
		if p == "" {
			return fmt.Errorf("nodejs: libnode.so missing for ABI %q", abi)
		}
		if err := a.AddNativeLib(p, "lib/"+abi+"/libnode.so"); err != nil {
			return fmt.Errorf("nodejs: inject libnode.so/%s: %w", abi, err)
		}
		pc := libcxxPaths[abi]
		if pc == "" {
			return fmt.Errorf("nodejs: libc++_shared.so missing for ABI %q", abi)
		}
		if err := a.AddNativeLib(pc, "lib/"+abi+"/libc++_shared.so"); err != nil {
			return fmt.Errorf("nodejs: inject libc++_shared.so/%s: %w", abi, err)
		}
	}

	count := 0
	skipDirs := map[string]bool{
		".git": true, "node_modules": true,
		".cache": true, ".next": true, ".nuxt": true, "__pycache__": true,
		".idea": true, ".vscode": true, ".gradle": true,
		"build": true, "dist": true, ".w2a-nodebuild": true,
	}
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(src, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "." {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".bak") || strings.HasSuffix(name, ".bak2") ||
			strings.HasSuffix(name, ".old") || strings.HasSuffix(name, ".swp") ||
			name == ".DS_Store" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := a.AddAssetStoredBytes("assets/nodejs_app/"+rel, data); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return fmt.Errorf("nodejs: copy source: %w", err)
	}

	envVars := cfg.NodeJS.EnvVars
	if envVars == nil {
		envVars = map[string]string{}
	}
	cfgJSON := map[string]any{
		"appType":   "NODEJS_APP",
		"language":  "RUSSIAN",
		"targetUrl": "nodejs://localhost",
		"nodejsConfig": map[string]any{
			"mode":                 "FULLSTACK",
			"port":                 cfg.NodeJS.Port,
			"portConflictMode":     "AUTO_KILL",
			"entryFile":            entry,
			"envVars":              envVars,
			"customNodeExtensions": []any{},
		},
	}
	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/app_config.json", data)

	fmt.Printf("nodejs: injected %d files → assets/nodejs_app/, %d ABI libs\n", count, len(abis))
	return a.SaveAPK(outPath)
}

func nodeCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "w2a", "nodejs", nodeVersionZip)
}

func ensureLibnodeCached(abis []string) (map[string]string, error) {
	dir := nodeCacheDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	out := make(map[string]string)
	missing := false
	for _, abi := range abis {
		p := filepath.Join(dir, "libnode-"+abi+".so")
		if st, err := os.Stat(p); err == nil && st.Size() > 0 {
			out[abi] = p
		} else {
			missing = true
			break
		}
	}
	if !missing {
		return out, nil
	}

	zipPath := filepath.Join(dir, "nodejs-mobile.zip")
	if st, err := os.Stat(zipPath); err != nil || st.Size() == 0 {
		fmt.Printf("nodejs: downloading nodejs-mobile %s (~55 MB)...\n", nodeVersionZip)
		if err := httpDownload(nodeZipURL, zipPath); err != nil {
			return nil, fmt.Errorf("download: %w", err)
		}
	}

	actual, err := sha256File(zipPath)
	if err != nil {
		return nil, err
	}
	if actual != nodeZipSHA256 {
		os.Remove(zipPath)
		return nil, fmt.Errorf("SHA256 mismatch: got %s want %s", actual, nodeZipSHA256)
	}
	fmt.Printf("nodejs: SHA256 OK\n")

	for _, abi := range abis {
		entry, ok := nodeZipPaths[abi]
		if !ok {
			return nil, fmt.Errorf("unsupported ABI %q", abi)
		}
		outPath := filepath.Join(dir, "libnode-"+abi+".so")
		if _, err := os.Stat(outPath); err == nil {
			out[abi] = outPath
			continue
		}
		fmt.Printf("nodejs: extracting %s\n", entry)
		if err := zipExtractOne(zipPath, entry, outPath); err != nil {
			return nil, fmt.Errorf("extract %s: %w", entry, err)
		}
		out[abi] = outPath
	}
	return out, nil
}

func resolveLibcxxPaths(abis []string) (map[string]string, error) {
	ndk := findNDK()
	if ndk == "" {
		return nil, fmt.Errorf("NDK not found — set ANDROID_NDK_HOME")
	}
	prebuilt := filepath.Join(ndk, "toolchains", "llvm", "prebuilt")
	entries, err := os.ReadDir(prebuilt)
	if err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("NDK prebuilt toolchain not found at %s", prebuilt)
	}
	host := entries[0].Name()
	out := make(map[string]string)
	for _, abi := range abis {
		tmpl, ok := ndkLibcxxPaths[abi]
		if !ok {
			return nil, fmt.Errorf("libc++_shared.so unknown for ABI %q", abi)
		}
		rel := strings.ReplaceAll(tmpl, "<host>", host)
		p := filepath.Join(ndk, rel)
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("libc++_shared.so not found: %s", p)
		}
		out[abi] = p
	}
	return out, nil
}

func findNDK() string {
	roots := []string{
		os.Getenv("ANDROID_NDK_HOME"),
		os.Getenv("ANDROID_NDK_ROOT"),
	}
	if v := os.Getenv("ANDROID_HOME"); v != "" {
		roots = append(roots, filepath.Join(v, "ndk"))
	}
	if v := os.Getenv("ANDROID_SDK_ROOT"); v != "" {
		roots = append(roots, filepath.Join(v, "ndk"))
	}
	for _, r := range roots {
		if r == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(r, "toolchains")); err == nil {
			return r
		}
		entries, err := os.ReadDir(r)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			sub := filepath.Join(r, e.Name())
			if _, err := os.Stat(filepath.Join(sub, "toolchains")); err == nil {
				return sub
			}
		}
	}
	return ""
}

func httpDownload(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, url)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func zipExtractOne(zipPath, entryName, dst string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != entryName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, rc)
		return err
	}
	return fmt.Errorf("entry %q not found in zip", entryName)
}
