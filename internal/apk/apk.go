package apk

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// APK is a convenience wrapper around a ZipArchive plus decoded manifest.
type APK struct {
	Archive  *ZipArchive
	Manifest *AXMLDocument
	// manifestDirty is true when the in-memory manifest was modified and
	// therefore needs to be re-serialised on save. If false, the original
	// AXML bytes are preserved (important while WriteAXML is being fixed —
	// any re-encode currently breaks aapt2).
	manifestDirty bool
}

// LoadAPK reads an APK file from disk, parses the AndroidManifest.xml.
func LoadAPK(path string) (*APK, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load apk: %w", err)
	}
	return LoadAPKBytes(data)
}

// LoadAPKBytes is like LoadAPK but takes raw bytes.
func LoadAPKBytes(data []byte) (*APK, error) {
	archive, err := LoadZip(data)
	if err != nil {
		return nil, err
	}
	manifestEntry := archive.Get("AndroidManifest.xml")
	if manifestEntry == nil {
		return nil, fmt.Errorf("apk: AndroidManifest.xml not found")
	}
	doc, err := ParseAXML(manifestEntry.Data)
	if err != nil {
		return nil, fmt.Errorf("apk: parse manifest: %w", err)
	}
	return &APK{Archive: archive, Manifest: doc}, nil
}

// SaveAPK writes the APK to a file, re-encoding the manifest first.
func (a *APK) SaveAPK(path string) error {
	if err := a.ReencodeManifest(); err != nil {
		return err
	}
	if n := a.stripSignatureArtifacts(); n > 0 {
		fmt.Printf("signature: stripped %d META-INF entries (v1 invalidated by rewrite)\n", n)
	}
	data, err := a.Archive.Save()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// SaveAPKBytes returns the final archive as bytes (with manifest re-encoded).
func (a *APK) SaveAPKBytes() ([]byte, error) {
	if err := a.ReencodeManifest(); err != nil {
		return nil, err
	}
	if n := a.stripSignatureArtifacts(); n > 0 {
		fmt.Printf("signature: stripped %d META-INF entries (v1 invalidated by rewrite)\n", n)
	}
	return a.Archive.Save()
}

// ReencodeManifest writes the current AXMLDocument tree back into
// AndroidManifest.xml inside the archive, using STORED method.
func (a *APK) ReencodeManifest() error {
	if a.Manifest == nil {
		return fmt.Errorf("apk: no manifest")
	}
	if !a.manifestDirty {
		// Nothing changed — keep original AXML bytes. Our WriteAXML is not
		// yet byte-compatible with aapt2; re-encoding corrupts the file.
		return nil
	}
	bytes, err := WriteAXML(a.Manifest)
	if err != nil {
		return fmt.Errorf("apk: write manifest: %w", err)
	}
	a.Archive.Replace("AndroidManifest.xml", bytes)
	return nil
}

// stripSignatureArtifacts removes v1 signing artifacts from META-INF/
// after the archive has been rewritten. When we re-pack the zip, all file
// hashes change, so any pre-existing MANIFEST.MF / *.SF / *.RSA / *.DSA /
// *.EC become invalid. v2/v3 signatures live in APK Signing Block (not
// in zip entries) and are silently dropped by Archive.Save() — no action
// needed for those.
//
// After stripping, the APK is unsigned. Caller must re-sign (wta sign or
// apksigner) before install.
func (a *APK) stripSignatureArtifacts() int {
	var toRemove []string
	for _, e := range a.Archive.Entries {
		if !strings.HasPrefix(e.Name, "META-INF/") {
			continue
		}
		base := strings.ToUpper(path.Base(e.Name))
		if strings.HasSuffix(base, ".SF") ||
			strings.HasSuffix(base, ".RSA") ||
			strings.HasSuffix(base, ".DSA") ||
			strings.HasSuffix(base, ".EC") ||
			base == "MANIFEST.MF" {
			toRemove = append(toRemove, e.Name)
		}
	}
	for _, n := range toRemove {
		a.Archive.Remove(n)
	}
	return len(toRemove)
}

// Package returns the manifest's `package` attribute.
func (a *APK) Package() string {
	if a.Manifest == nil || a.Manifest.Root == nil {
		return ""
	}
	return a.Manifest.Root.AttrValue("package")
}

// SetPackage rewrites the manifest's `package` attribute.
func (a *APK) SetPackage(pkg string) {
	if a.Manifest == nil || a.Manifest.Root == nil {
		return
	}
	a.Manifest.Root.SetAttr("package", pkg)
	a.manifestDirty = true
}

// VersionCode returns the manifest's `android:versionCode` as a uint32, or 0.
func (a *APK) VersionCode() uint32 {
	if a.Manifest == nil || a.Manifest.Root == nil {
		return 0
	}
	attr := a.Manifest.Root.Attr("versionCode")
	if attr == nil {
		return 0
	}
	return attr.ValueData
}

// SetVersionCode sets the manifest's versionCode (integer type).
func (a *APK) SetVersionCode(code uint32) {
	if a.Manifest == nil || a.Manifest.Root == nil {
		return
	}
	if a.VersionCode() == code {
		return // no-op; avoid WriteAXML (corrupts manifest)
	}
	a.manifestDirty = true
	root := a.Manifest.Root
	// Remove existing then append with proper integer type.
	for i := range root.Attributes {
		if root.Attributes[i].Name == "versionCode" {
			root.Attributes[i].ValueType = TypeIntDec
			root.Attributes[i].ValueData = code
			root.Attributes[i].RawValue = ""
			root.Attributes[i].ValueStr = ""
			return
		}
	}
	root.Attributes = append(root.Attributes, AXMLAttribute{
		Name:      "versionCode",
		ValueType: TypeIntDec,
		ValueData: code,
	})
}

// VersionName returns the manifest's `android:versionName`.
func (a *APK) VersionName() string {
	if a.Manifest == nil || a.Manifest.Root == nil {
		return ""
	}
	return a.Manifest.Root.AttrValue("versionName")
}

// SetVersionName sets the manifest's versionName.
func (a *APK) SetVersionName(name string) {
	if a.Manifest == nil || a.Manifest.Root == nil {
		return
	}
	if a.VersionName() == name {
		return // no-op; avoid WriteAXML (corrupts manifest)
	}
	a.Manifest.Root.SetAttr("versionName", name)
	a.manifestDirty = true
}

// MarkManifestDirty records that the manifest was modified externally
// (e.g. via AddPermission) and needs re-encoding on save.
func (a *APK) MarkManifestDirty() { a.manifestDirty = true }
