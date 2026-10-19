package apkutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InjectIntoAPK adds entries to an existing APK (zip update).
// Uses the `zip` CLI tool because Go's archive/zip is read-only.
// entries: map of entryName (e.g. "classes.dex", "lib/arm64-v8a/libfoo.so")
// to absolute source path on disk.
func InjectIntoAPK(apkPath string, entries map[string]string) error {
	if _, err := os.Stat(apkPath); err != nil {
		return fmt.Errorf("apk not found: %s", apkPath)
	}
	zipBin, err := exec.LookPath("zip")
	if err != nil {
		return fmt.Errorf("zip not found in PATH: %w", err)
	}

	// group by target dir so `zip` runs with correct relative paths
	type item struct{ entry, src string }
	var toAdd []item
	for entry, src := range entries {
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("inject source not found: %s (%s)", src, entry)
		}
		toAdd = append(toAdd, item{entry, src})
	}
	// stable order
	for i := 0; i < len(toAdd); i++ {
		for j := i + 1; j < len(toAdd); j++ {
			if toAdd[i].entry > toAdd[j].entry {
				toAdd[i], toAdd[j] = toAdd[j], toAdd[i]
			}
		}
	}

	// create a staging tree so paths inside zip are correct
	tmpDir, err := os.MkdirTemp("", "apkinject-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, it := range toAdd {
		dst := filepath.Join(tmpDir, it.entry)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := copyFile(it.src, dst); err != nil {
			return err
		}
	}

	// zip -u <apk> <files...> from tmpDir
	var files []string
	for _, it := range toAdd {
		files = append(files, it.entry)
	}
	args := append([]string{"-u", apkPath}, files...)
	cmd := exec.Command(zipBin, args...)
	cmd.Dir = tmpDir
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("zip -u failed: %v\n%s", err, buf.String())
	}
	return nil
}

// CollectJNILibs walks the given dirs and returns map entry->srcPath,
// where entry is "lib/<abi>/<filename>.so".
// Expects layout: <dir>/<abi>/*.so
func CollectJNILibs(dirs []string, wantedABIs []string) (map[string]string, error) {
	out := map[string]string{}
	want := map[string]bool{}
	for _, a := range wantedABIs {
		want[a] = true
	}
	for _, d := range dirs {
		abis, err := os.ReadDir(d)
		if err != nil {
			return nil, fmt.Errorf("jniLibs dir %s: %w", d, err)
		}
		for _, abiEntry := range abis {
			if !abiEntry.IsDir() {
				continue
			}
			abi := abiEntry.Name()
			if len(want) > 0 && !want[abi] {
				continue
			}
			abiDir := filepath.Join(d, abi)
			files, err := os.ReadDir(abiDir)
			if err != nil {
				return nil, err
			}
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".so") {
					continue
				}
				entry := "lib/" + abi + "/" + f.Name()
				out[entry] = filepath.Join(abiDir, f.Name())
			}
		}
	}
	return out, nil
}

// CollectJavaSources returns list of absolute .java file paths under dirs.
func CollectJavaSources(dirs []string) ([]string, error) {
	var out []string
	for _, d := range dirs {
		err := filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".java") {
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", d, err)
		}
	}
	return out, nil
}

// Javac compiles sources with given classpath into outDir.
func Javac(javaRelease, classpath, outDir string, sources []string) error {
	if len(sources) == 0 {
		return fmt.Errorf("no java sources")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	args := []string{"--release", javaRelease}
	if classpath != "" {
		args = append(args, "-classpath", classpath)
	}
	args = append(args, "-d", outDir)
	args = append(args, sources...)
	cmd := exec.Command("javac", args...)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("javac failed: %v\n%s", err, buf.String())
	}
	return nil
}

// JarClasses packs classesDir into a jar (needed for d8 which prefers jars).
func JarClasses(classesDir, outJar string) error {
	args := []string{"cf", outJar, "-C", classesDir, "."}
	cmd := exec.Command("jar", args...)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("jar failed: %v\n%s", err, buf.String())
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
