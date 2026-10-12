package arsc

import (
	"archive/zip"
	"io"
	"testing"
)

func TestAddStringEntry(t *testing.T) {
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
		t.Fatal(err)
	}
	if len(table.Packages) == 0 {
		t.Skip("no packages")
	}

	// Find a type ID present.
	var tid uint8
	for id := range table.Packages[0].Types {
		tid = id
		break
	}
	t.Logf("using typeID=0x%02X", tid)

	// Count entries before.
	before := table.Packages[0].Types[tid][0].EntryCount
	t.Logf("entries before: %d", before)

	// Add a new entry.
	idx := table.AddStringEntry(0x7F, tid, "WTA_ADDED_ENTRY")
	if idx < 0 {
		t.Fatal("AddStringEntry failed")
	}
	t.Logf("added at index: %d", idx)

	// Roundtrip.
	encoded := WriteTable(table)
	parsed, err := ParseTable(encoded)
	if err != nil {
		t.Fatal(err)
	}
	after := parsed.Packages[0].Types[tid][0].EntryCount
	t.Logf("entries after:  %d", after)
	if after != before+1 {
		t.Errorf("entry count: got %d, want %d", after, before+1)
	}

	// Verify entry resolves.
	e := parsed.FindEntry(0x7F, tid, uint32(idx))
	if e == nil || e.Value == nil {
		t.Fatal("new entry not found")
	}
	resolved := parsed.ResolveString(e.Value.Data)
	t.Logf("resolved: %q", resolved)
	if resolved != "WTA_ADDED_ENTRY" {
		t.Errorf("resolved: got %q", resolved)
	}
}
