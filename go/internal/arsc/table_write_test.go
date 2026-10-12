package arsc

import (
	"archive/zip"
	"io"
	"testing"
)

func TestFullTableRoundtrip(t *testing.T) {
	apkPath := findAnyAPK()
	if apkPath == "" {
		t.Skip("no test APK")
	}
	zr, _ := zip.OpenReader(apkPath)
	defer zr.Close()

	var rc io.ReadCloser
	for _, f := range zr.File {
		if f.Name == "resources.arsc" {
			rc, _ = f.Open()
			break
		}
	}
	if rc == nil {
		t.Skip("no resources.arsc")
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)

	original, err := ParseTable(data)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("original: %d bytes, %d packages, pool=%d",
		len(data), len(original.Packages), len(original.GlobalPool.Strings))

	// Serialize.
	encoded := WriteTable(original)
	t.Logf("encoded: %d bytes", len(encoded))

	// Parse back.
	parsed, err := ParseTable(encoded)
	if err != nil {
		t.Fatalf("ParseTable(encoded): %v", err)
	}
	t.Logf("parsed: %d packages, pool=%d",
		len(parsed.Packages), len(parsed.GlobalPool.Strings))

	if len(parsed.Packages) != len(original.Packages) {
		t.Errorf("package count: got %d, want %d",
			len(parsed.Packages), len(original.Packages))
	}
	if len(parsed.GlobalPool.Strings) != len(original.GlobalPool.Strings) {
		t.Errorf("pool size: got %d, want %d",
			len(parsed.GlobalPool.Strings), len(original.GlobalPool.Strings))
	}

	// Verify a few strings.
	for i := 0; i < 5 && i < len(original.GlobalPool.Strings); i++ {
		if parsed.GlobalPool.Strings[i] != original.GlobalPool.Strings[i] {
			t.Errorf("string[%d]: got %q, want %q",
				i, parsed.GlobalPool.Strings[i], original.GlobalPool.Strings[i])
		}
	}

	// Verify package name.
	if len(parsed.Packages) > 0 && len(original.Packages) > 0 {
		if parsed.Packages[0].Name != original.Packages[0].Name {
			t.Errorf("package name: got %q, want %q",
				parsed.Packages[0].Name, original.Packages[0].Name)
		}
		if parsed.Packages[0].ID != original.Packages[0].ID {
			t.Errorf("package ID: got 0x%02X, want 0x%02X",
				parsed.Packages[0].ID, original.Packages[0].ID)
		}
	}

	// Verify an entry resolves to the same string.
	e1 := original.FindEntry(0x7F, 0x0F, 1)
	e2 := parsed.FindEntry(0x7F, 0x0F, 1)
	if e1 != nil && e2 != nil && e1.Value != nil && e2.Value != nil {
		s1 := original.ResolveString(e1.Value.Data)
		s2 := parsed.ResolveString(e2.Value.Data)
		if s1 != s2 {
			t.Errorf("entry[0x0F/1] resolve: got %q, want %q", s2, s1)
		}
	}
}
