package apk

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IconEntry describes one icon file found in the archive.
type IconEntry struct {
	Name   string
	Size   int
	Folder string // "mipmap-hdpi" etc.
}

// FindIcons returns all ic_launcher-like entries in the APK.
func (a *APK) FindIcons() []IconEntry {
	var out []IconEntry
	for _, e := range a.Archive.Entries {
		lower := strings.ToLower(e.Name)
		if !strings.HasSuffix(lower, ".png") {
			continue
		}
		// Only mipmap-* folders and only files named ic_launcher*.
		base := lower
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		if !strings.HasPrefix(base, "ic_launcher") {
			continue
		}
		if !strings.HasPrefix(lower, "res/mipmap") {
			continue
		}
		folder := ""
		if idx := strings.LastIndex(e.Name, "/"); idx > 0 {
			parts := strings.Split(e.Name, "/")
			if len(parts) >= 2 {
				folder = parts[1]
			}
		}
		out = append(out, IconEntry{
			Name:   e.Name,
			Size:   len(e.Data),
			Folder: folder,
		})
	}
	return out
}

// ReplaceIcon replaces every ic_launcher-like PNG with the given file.
// Returns the number of entries replaced.
func (a *APK) ReplaceIcon(pngPath string) (int, error) {
	data, err := os.ReadFile(pngPath)
	if err != nil {
		return 0, fmt.Errorf("apk: read icon: %w", err)
	}
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return 0, fmt.Errorf("apk: not a PNG file")
	}
	if len(data) > 1024*1024 {
		return 0, fmt.Errorf("apk: PNG too large (%d bytes)", len(data))
	}
	_ = filepath.Base(pngPath)

	icons := a.FindIcons()
	count := 0
	for _, icon := range icons {
		a.Archive.ReplaceDeflated(icon.Name, data)
		count++
	}
	return count, nil
}
