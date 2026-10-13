// Package assetsync copies build outputs into app/src/main/assets/.
package assetsync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// CopyFile copies src to dst, creating parent directories as needed.
func CopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
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

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// CopyMatching copies every file matching the glob pattern into dstDir
// preserving basenames. Returns the number of files copied.
func CopyMatching(globPattern, dstDir string) (int, error) {
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return 0, err
	}
	if len(matches) == 0 {
		return 0, fmt.Errorf("no files match %s", globPattern)
	}
	sort.Strings(matches)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return 0, err
	}
	count := 0
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		dst := filepath.Join(dstDir, filepath.Base(m))
		if err := CopyFile(m, dst); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
