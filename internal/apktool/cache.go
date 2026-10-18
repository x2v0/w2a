package apktool

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/cache"
)

// cacheRoot returns ~/.cache/w2a/apktool-d
func cacheRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "w2a", "apktool-d")
}

// getDecompiled returns path to cached src/, or "" if absent.
// Uses internal/cache.HashBytes for fingerprint. Checks .complete marker.
func getDecompiled(templateBytes []byte) string {
	fp := cache.HashBytes(templateBytes)
	dir := filepath.Join(cacheRoot(), string(fp))
	if _, err := os.Stat(filepath.Join(dir, ".complete")); err != nil {
		return ""
	}
	src := filepath.Join(dir, "src")
	if _, err := os.Stat(src); err != nil {
		return ""
	}
	return src
}

// saveDecompiled copies srcDir into cache/<fp>/src + .complete marker.
// Atomic: writes to .tmp, then renames.
func saveDecompiled(templateBytes []byte, srcDir string) error {
	fp := cache.HashBytes(templateBytes)
	dir := filepath.Join(cacheRoot(), string(fp))
	tmp := dir + ".tmp"

	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}

	if err := copyTree(srcDir, filepath.Join(tmp, "src")); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}

	if err := os.WriteFile(filepath.Join(tmp, ".complete"), []byte("ok\n"), 0o644); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}

	_ = os.RemoveAll(dir)
	if err := os.Rename(tmp, dir); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	return nil
}

// copyTree recursively copies src → dst (dirs, files, perms).
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		return err
	})
}

// CleanCache removes the entire apktool-d cache.
func CleanCache() error {
	dir := cacheRoot()
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "apktool: cache cleaned (%s)\n", dir)
	return nil
}

// CacheSize returns total size of apktool-d cache in bytes.
func CacheSize() int64 {
	var total int64
	_ = filepath.Walk(cacheRoot(), func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}
