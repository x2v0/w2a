// Package cache implements input fingerprinting for incremental builds.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Fingerprint is a hex-encoded SHA-256 over a set of files.
type Fingerprint string

// HashFiles hashes each file's path + size + mtime + content. Files are
// visited in sorted order for determinism. Directories are recursed.
//
// Returns an error if any file is unreadable (which invalidates the cache
// for the caller).
func HashFiles(paths []string) (Fingerprint, error) {
	var collected []string

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			err := filepath.WalkDir(p, func(sub string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					collected = append(collected, sub)
				}
				return nil
			})
			if err != nil {
				return "", err
			}
		} else {
			collected = append(collected, p)
		}
	}
	sort.Strings(collected)

	h := sha256.New()
	for _, path := range collected {
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		// Metadata: path, size, modtime.
		fmt.Fprintf(h, "%s\x00%d\x00%d\x00", path, info.Size(), info.ModTime().UnixNano())

		// Content (with length prefix to avoid collisions).
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%d\x00", info.Size())
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
		h.Write([]byte{0})
	}
	return Fingerprint(hex.EncodeToString(h.Sum(nil))), nil
}

// HashBytes hashes raw bytes as a fingerprint.
func HashBytes(data []byte) Fingerprint {
	h := sha256.Sum256(data)
	return Fingerprint(hex.EncodeToString(h[:]))
}

// Empty is the zero fingerprint.
const Empty Fingerprint = ""
