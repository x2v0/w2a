package arsc

import (
	"archive/zip"
	"io"
	"testing"
)

func TestFullTable(t *testing.T) {
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

	table, err := ParseTable(data)
	if err != nil {
		t.Fatalf("ParseTable: %v", err)
	}
	t.Logf("global pool: %d strings", len(table.GlobalPool.Strings))
	t.Logf("packages: %d", len(table.Packages))

	for _, pkg := range table.Packages {
		t.Logf("  package id=0x%02X name=%q", pkg.ID, pkg.Name)
		for typeID, list := range pkg.Types {
			for _, tt := range list {
				t.Logf("    type[0x%02X] entries=%d config=%s",
					typeID, tt.EntryCount, tt.Config)
			}
		}
	}

	e := table.FindEntry(0x7F, 0x0F, 1)
	if e != nil && e.Value != nil {
		t.Logf("entry[0x0F/1]: type=0x%02X data=0x%08X", e.Value.DataType, e.Value.Data)
		if e.Value.DataType == TypeString {
			t.Logf("  resolved: %q", table.ResolveString(e.Value.Data))
		}
	}
}
