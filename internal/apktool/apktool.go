// Package apktool wraps apktool (d/b) to edit AndroidManifest.xml as plain XML.
//
// Used because our in-tree AXML writer (internal/apk/axml_write.go) is not
// byte-compatible with aapt2 — re-encoding corrupts the manifest (invalid
// chunk, verify fails). apktool d/b is the battle-tested fallback.
//
// Requires: `apktool` + `java` in PATH (pkg install apktool openjdk-21).
package apktool

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ManifestPatch describes the edits to apply to AndroidManifest.xml.
type ManifestPatch struct {
	Package     string   // new package (empty = keep)
	VersionCode *uint32  // new versionCode (nil = keep)
	VersionName *string  // new versionName (nil = keep)
	AppLabel    string   // <application android:label="..."> (empty = keep)
	AddPerms    []string // <uses-permission> to add (deduped)
	RemovePerms []string // <uses-permission> to remove (by android:name)
}

// Empty returns true if the patch has no changes.
func (p ManifestPatch) Empty() bool {
	return p.Package == "" &&
		p.VersionCode == nil &&
		p.VersionName == nil &&
		p.AppLabel == "" &&
		len(p.AddPerms) == 0 &&
		len(p.RemovePerms) == 0
}

// Available reports whether apktool is in PATH.
func Available() bool {
	_, err := exec.LookPath("apktool")
	return err == nil
}

// Repack decompiles apk, applies patch to AndroidManifest.xml, recompiles,
// and returns the new APK bytes (unsigned — caller must sign).
// Repack decompiles apk, applies patch to AndroidManifest.xml, recompiles.
//
// cacheKey is used for the decompile cache fingerprint. Pass the ORIGINAL
// template bytes (stable) — apkBytes may include injected assets or volatile
// zip timestamps that would invalidate the cache every build.
// If cacheKey == nil, apkBytes is used.
func Repack(apkBytes []byte, patch ManifestPatch, cacheKey []byte) ([]byte, error) {
	if !Available() {
		return nil, fmt.Errorf("apktool: not in PATH (pkg install apktool openjdk-21)")
	}
	if patch.Empty() {
		return apkBytes, nil
	}
	if cacheKey == nil {
		cacheKey = apkBytes
	}

	// tmp dir for apktool work
	tmpDir, err := os.MkdirTemp("", "wta-apktool-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	inPath := filepath.Join(tmpDir, "in.apk")
	workDir := filepath.Join(tmpDir, "src")
	outPath := filepath.Join(tmpDir, "out.apk")

	if err := os.WriteFile(inPath, apkBytes, 0o600); err != nil {
		return nil, err
	}

	// 1. Decompile (with cache)
	// FIXME: cache disabled — see cache.go (key collision bug)
	if err := runStream("apktool", "d", "-f", "-q", "-o", workDir, inPath); err != nil {
		return nil, fmt.Errorf("apktool d: %w", err)
	}

	// 1.5 Fix apktool artifacts: $ in AVD file names, references.
	// aapt2 2.20+ rejects $ in resource names ('invalid entry name').
	// apktool d emits $avd_*__*.xml — we rename to _avd_*__*.xml
	// and rewrite @drawable/$* refs to @drawable/_*.
	if err := fixAvdNames(workDir); err != nil {
		return nil, fmt.Errorf("fix AVD names: %w", err)
	}

	// 2. Patch AndroidManifest.xml
	manifestPath := filepath.Join(workDir, "AndroidManifest.xml")
	if err := patchManifestFile(manifestPath, patch); err != nil {
		return nil, fmt.Errorf("patch manifest: %w", err)
	}

	// 3. Recompile. apktool 3.x only supports aapt2 (no --use-aapt1).
	// If this fails, the caller should fall back to in-place (SetPackageInPlace).
	if err := runStream("apktool", "b", "-q", "--no-crunch", "-o", outPath, workDir); err != nil {
		return nil, fmt.Errorf("apktool b: %w", err)
	}

	// 4. Read result
	out, err := os.ReadFile(outPath)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// runStream runs a command and streams its output to os.Stderr on error.
func runStream(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Keep tail of output — apktool prints a lot
		s := string(out)
		if len(s) > 2000 {
			s = s[len(s)-2000:]
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n%s\n", name, err, s)
		return err
	}
	return nil
}

// patchManifestFile edits plain-text AndroidManifest.xml in place.
//
// We use encoding/xml with struct capture — preserves unknown elements and
// namespaces (apktool emits a clean xmlns:android="..."). Patch semantics:
//   - package: set root attr
//   - versionCode/versionName: set android: attrs (root)
//   - AddPerms: append <uses-permission android:name="..."/> (deduped)
//   - RemovePerms: delete <uses-permission> whose android:name matches
func patchManifestFile(path string, p ManifestPatch) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse into a tree (generic xml tokens → mini DOM).
	doc, err := parseGeneric(data)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if p.Package != "" {
		doc.root.attrs["package"] = p.Package
	}
	if p.VersionCode != nil {
		doc.root.attrs["android:versionCode"] = fmt.Sprintf("%d", *p.VersionCode)
	}
	if p.VersionName != nil {
		doc.root.attrs["android:versionName"] = *p.VersionName
	}

	// App label: set android:label on <application>
	if p.AppLabel != "" {
		var found bool
		for _, c := range doc.root.children {
			if c.name == "application" {
				c.attrs["android:label"] = p.AppLabel
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "apktool: warn: <application> not found, label not set\n")
		}
	}

	// Remove perms
	if len(p.RemovePerms) > 0 {
		rm := map[string]bool{}
		for _, n := range p.RemovePerms {
			rm[n] = true
			if !strings.HasPrefix(n, "android.permission.") {
				rm["android.permission."+n] = true
			}
		}
		var kept []*node
		for _, c := range doc.root.children {
			if c.name == "uses-permission" {
				name := c.attrs["android:name"]
				if rm[name] {
					continue
				}
			}
			kept = append(kept, c)
		}
		doc.root.children = kept
	}

	// Add perms
	if len(p.AddPerms) > 0 {
		existing := map[string]bool{}
		for _, c := range doc.root.children {
			if c.name == "uses-permission" {
				existing[c.attrs["android:name"]] = true
			}
		}
		for _, n := range p.AddPerms {
			full := n
			if !strings.HasPrefix(n, "android.permission.") &&
				!strings.HasPrefix(n, "com.") &&
				!strings.HasPrefix(n, "org.") {
				full = "android.permission." + n
			}
			if existing[full] {
				continue
			}
			doc.root.children = append(doc.root.children, &node{
				name:  "uses-permission",
				attrs: map[string]string{"android:name": full},
			})
			existing[full] = true
		}
	}

	// Write back
	out := &strings.Builder{}
	out.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	doc.write(out, 0)
	return os.WriteFile(path, []byte(out.String()), 0o644)
}

// --- mini DOM ---

type node struct {
	name     string
	attrs    map[string]string
	children []*node
}

type document struct {
	root *node
}

func parseGeneric(data []byte) (*document, error) {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	dec.Strict = false

	var root *node
	var stack []*node
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{name: t.Name.Local, attrs: map[string]string{}}
			for _, a := range t.Attr {
				key := a.Name.Local
				if a.Name.Space != "" {
					if a.Name.Space == "http://schemas.android.com/apk/res/android" {
						key = "android:" + key
					} else {
						key = a.Name.Space + ":" + key
					}
				}
				n.attrs[key] = a.Value
			}
			if root == nil {
				root = n
			} else if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("no root element")
	}
	return &document{root: root}, nil
}

func (d *document) write(sb *strings.Builder, depth int) {
	d.root.write(sb, depth)
}

func (n *node) write(sb *strings.Builder, depth int) {
	indent := strings.Repeat("    ", depth)
	sb.WriteString(indent)
	sb.WriteString("<")
	sb.WriteString(n.name)
	// stable attr order
	keys := make([]string, 0, len(n.attrs))
	for k := range n.attrs {
		keys = append(keys, k)
	}
	// crude sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		sb.WriteString(" ")
		sb.WriteString(attrName(k))
		sb.WriteString("=\"")
		sb.WriteString(xmlEscape(n.attrs[k]))
		sb.WriteString("\"")
	}
	if len(n.children) == 0 {
		sb.WriteString(" />\n")
		return
	}
	sb.WriteString(">\n")
	for _, c := range n.children {
		c.write(sb, depth+1)
	}
	sb.WriteString(indent)
	sb.WriteString("</")
	sb.WriteString(n.name)
	sb.WriteString(">\n")
}

func attrName(k string) string {
	// Android namespace attrs already prefixed with android:
	return k
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// fixAvdNames renames $*.xml in res/drawable/ to _*.xml and rewrites
// @drawable/$* references to @drawable/_* in all res/*.xml.
//
// aapt2 2.20+ (Android 16 SDK) rejects resource names with '$'
// ('invalid entry name'). apktool 3.x still emits them for AnimatedVectorDrawables
// (Material Components password toggle / checkbox / switch animations).
//
// AndroidManifest.xml references are unaffected (uses-permission, etc).
func fixAvdNames(workDir string) error {
	resDir := filepath.Join(workDir, "res")
	drawableDir := filepath.Join(resDir, "drawable")

	entries, err := os.ReadDir(drawableDir)
	if err != nil {
		// no drawable dir — nothing to do (not fatal)
		return nil
	}

	renamed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "$") {
			continue
		}
		newName := "_" + strings.TrimPrefix(name, "$")
		oldPath := filepath.Join(drawableDir, name)
		newPath := filepath.Join(drawableDir, newName)
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("rename %s: %w", name, err)
		}
		renamed++
	}
	if renamed > 0 {
		fmt.Fprintf(os.Stderr, "apktool: renamed %d \\$*.xml → _*.xml in res/drawable/\n", renamed)
	}

	// Rewrite every '$' in res/*.xml to '_'.
	// - @drawable/$avd_xxx → @drawable/_avd_xxx (references)
	// - public.xml: name="$avd_xxx" → name="_avd_xxx" (declarations)
	// apktool d emits both; aapt2 2.20+ rejects '$' in resource names.
	refsFixed := 0
	err = filepath.Walk(resDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".xml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		orig := string(data)
		fixed := strings.ReplaceAll(orig, "$", "_")
		if fixed != orig {
			if err := os.WriteFile(path, []byte(fixed), 0o644); err != nil {
				return err
			}
			refsFixed++
		}
		return nil
	})
	if err != nil {
		return err
	}
	if refsFixed > 0 {
		fmt.Fprintf(os.Stderr, "apktool: fixed @drawable/\\$ refs in %d files\n", refsFixed)
	}
	return nil
}
