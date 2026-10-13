package apk

import (
	"fmt"
	"os"
	"strings"
)

// ListAssets returns names of all entries under assets/.
func (a *APK) ListAssets() []string {
	var out []string
	for _, e := range a.Archive.Entries {
		if strings.HasPrefix(e.Name, "assets/") {
			out = append(out, e.Name)
		}
	}
	return out
}

// AddAsset adds a local file as an APK asset entry at dstName.
// dstName must start with "assets/".
func (a *APK) AddAsset(srcPath, dstName string) error {
	if !strings.HasPrefix(dstName, "assets/") {
		return fmt.Errorf("apk: dst must start with assets/, got %q", dstName)
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("apk: read asset: %w", err)
	}
	a.Archive.ReplaceDeflated(dstName, data)
	return nil
}

// AddAssetBytes is like AddAsset but takes raw bytes.
func (a *APK) AddAssetBytes(dstName string, data []byte) error {
	if !strings.HasPrefix(dstName, "assets/") {
		return fmt.Errorf("apk: dst must start with assets/, got %q", dstName)
	}
	a.Archive.ReplaceDeflated(dstName, data)
	return nil
}

// RemoveAsset deletes an entry under assets/. Returns true if removed.
func (a *APK) RemoveAsset(name string) bool {
	if !strings.HasPrefix(name, "assets/") {
		return false
	}
	if !a.Archive.Has(name) {
		return false
	}
	a.Archive.Remove(name)
	return true
}
