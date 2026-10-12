package apk

import (
	"fmt"
	"os"
	"strings"
)

// ListNativeLibs returns all entries under lib/<abi>/.
func (a *APK) ListNativeLibs() map[string][]string {
	out := map[string][]string{}
	for _, e := range a.Archive.Entries {
		if !strings.HasPrefix(e.Name, "lib/") {
			continue
		}
		parts := strings.SplitN(e.Name, "/", 3)
		if len(parts) != 3 {
			continue
		}
		abi := parts[1]
		out[abi] = append(out[abi], e.Name)
	}
	return out
}

// AddNativeLib adds a .so file to lib/<abi>/<name>.
// The file is stored UNCOMPRESSED — required for Android's dynamic linker
// and (on Android 15+) 16 KB page alignment. Call ZipalignFile via the
// Toolchain after SaveAPK to align offsets.
func (a *APK) AddNativeLib(srcPath, dstName string) error {
	if !strings.HasPrefix(dstName, "lib/") || !strings.HasSuffix(dstName, ".so") {
		return fmt.Errorf("apk: dst must be lib/<abi>/<name>.so, got %q", dstName)
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("apk: read .so: %w", err)
	}
	if len(data) < 4 || data[0] != 0x7F || data[1] != 'E' || data[2] != 'L' || data[3] != 'F' {
		return fmt.Errorf("apk: not an ELF file: %s", srcPath)
	}
	// Must be STORED (uncompressed) so the linker can mmap it.
	a.Archive.Replace(dstName, data)
	return nil
}

// RemoveNativeLib deletes a lib/<abi>/<name>.so entry.
func (a *APK) RemoveNativeLib(name string) bool {
	if !strings.HasPrefix(name, "lib/") {
		return false
	}
	if !a.Archive.Has(name) {
		return false
	}
	a.Archive.Remove(name)
	return true
}
