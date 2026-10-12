package arsc

import (
	"archive/zip"
	"io"
	"testing"
)

func TestModifyString(t *testing.T) {
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

	// Original string at index 0.
	orig := table.GlobalPool.Strings[0]
	t.Logf("string[0] before: %q", orig)

	// Modify it.
	newValue := "WTA-REPLACED-STRING-🤖"
	if !table.SetString(0, newValue) {
		t.Fatal("SetString failed")
	}

	// Serialize and re-parse.
	encoded := WriteTable(table)
	parsed, err := ParseTable(encoded)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.GlobalPool.Strings[0]
	t.Logf("string[0] after:  %q", got)
	if got != newValue {
		t.Errorf("modify roundtrip: got %q, want %q", got, newValue)
	}
}

func TestSetEntryString(t *testing.T) {
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

	// Try to modify entry [0x7F, 0x0F, 1].
	origEntry := table.FindEntry(0x7F, 0x0F, 1)
	if origEntry == nil || origEntry.Value == nil {
		t.Skip("no entry to modify")
	}
	t.Logf("entry[0x0F/1] before: type=0x%02X data=%d",
		origEntry.Value.DataType, origEntry.Value.Data)

	newValue := "Изменённая строка"
	if !table.SetEntryString(0x7F, 0x0F, 1, newValue) {
		t.Fatal("SetEntryString failed")
	}

	encoded := WriteTable(table)
	parsed, err := ParseTable(encoded)
	if err != nil {
		t.Fatal(err)
	}

	// Resolve again.
	after := parsed.FindEntry(0x7F, 0x0F, 1)
	if after == nil || after.Value == nil {
		t.Fatal("lost entry after roundtrip")
	}
	resolved := parsed.ResolveString(after.Value.Data)
	t.Logf("entry[0x0F/1] after:  %q", resolved)
	if resolved != newValue {
		t.Errorf("resolved: got %q, want %q", resolved, newValue)
	}
}
