package apk

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// findShellTemplate locates the shell APK in the repo, returns "" if not present.
func findShellTemplate() string {
	candidates := []string{
		"../../../app/src/main/assets/template/webview_shell.apk",
		"../../app/src/main/assets/template/webview_shell.apk",
		"app/src/main/assets/template/webview_shell.apk",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func TestParseRealManifest(t *testing.T) {
	apkPath := findShellTemplate()
	if apkPath == "" {
		t.Skip("shell-template.apk not found — skipping")
	}

	zr, err := zip.OpenReader(apkPath)
	if err != nil {
		t.Fatalf("open apk: %v", err)
	}
	defer zr.Close()

	var manifestReader io.ReadCloser
	for _, f := range zr.File {
		if f.Name == "AndroidManifest.xml" {
			manifestReader, err = f.Open()
			if err != nil {
				t.Fatalf("open manifest: %v", err)
			}
			break
		}
	}
	if manifestReader == nil {
		t.Fatal("AndroidManifest.xml not found in APK")
	}
	defer manifestReader.Close()

	data, err := io.ReadAll(manifestReader)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("manifest size: %d bytes", len(data))

	doc, err := ParseAXML(data)
	if err != nil {
		t.Fatalf("ParseAXML: %v", err)
	}
	if doc.Root == nil {
		t.Fatal("nil root")
	}

	t.Logf("root: %s", doc.Root.Name)
	if doc.Root.Name != "manifest" {
		t.Errorf("expected root 'manifest', got %q", doc.Root.Name)
	}

	pkg := doc.Root.AttrValue("package")
	t.Logf("package: %q", pkg)
	if pkg == "" {
		t.Error("package attribute missing")
	}

	app := doc.Root.FindChild("application")
	if app == nil {
		t.Fatal("no <application> child")
	}
	t.Logf("application attrs: %d", len(app.Attributes))

	// Find launcher activity
	activity := app.FindChild("activity")
	if activity != nil {
		t.Logf("first activity: %s", activity.AttrValue("name"))
	}

	_ = filepath.Base(apkPath)
}
