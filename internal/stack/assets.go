package stack

import (
	"fmt"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
)

// InjectAssets copies bundle assets into the target APK. Assets keep their
// full path (already prefixed with "assets/" in the bundle). Optionally
// prefix them with a subdir to namespace multiple stacks.
//
// Returns the number of assets injected.
func InjectAssets(target *apk.APK, b *Bundle, namespace string) (int, error) {
	n := 0
	for name, data := range b.Assets {
		dst := name
		if namespace != "" && strings.HasPrefix(name, "assets/") {
			rest := strings.TrimPrefix(name, "assets/")
			dst = "assets/" + strings.Trim(namespace, "/") + "/" + rest
		}
		if err := target.AddAssetBytes(dst, data); err != nil {
			return n, fmt.Errorf("stack: inject asset %q: %w", dst, err)
		}
		n++
	}
	return n, nil
}

// InjectRes copies bundle resource files (res/**) verbatim into the target
// APK. NOTE: this does NOT merge the stack.arsc resource table — the caller
// must ensure the stack's package ID is registered in the target (usually
// via `aapt2 link --package-id 0x80`). Without ARSC merge, these resources
// are not resolvable at runtime; the assets path is preferred.
func InjectRes(target *apk.APK, b *Bundle) (int, error) {
	n := 0
	for name, data := range b.ResFiles {
		// Store as STORED or DEFLATED? Compressed is fine for images.
		target.Archive.ReplaceDeflated(name, data)
		n++
	}
	return n, nil
}

// InjectNativeLibs copies bundle lib/<abi>/*.so into the target APK.
// Native libraries are stored UNCOMPRESSED (required by the Android
// dynamic linker). Returns the number of libs injected.
func InjectNativeLibs(target *apk.APK, b *Bundle) (int, error) {
	n := 0
	for name, data := range b.LibFiles {
		// Validate ELF magic.
		if len(data) < 4 || data[0] != 0x7F || data[1] != 'E' || data[2] != 'L' || data[3] != 'F' {
			continue
		}
		target.Archive.Replace(name, data) // STORED
		n++
	}
	return n, nil
}
