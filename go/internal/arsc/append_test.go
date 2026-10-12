package arsc

import (
	"archive/zip"
	"io"
	"testing"
)

func TestAppendString(t *testing.T) {
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
	before := len(table.GlobalPool.Strings)
	t.Logf("pool before: %d", before)

	// Add two new strings.
	i1 := table.AppendString("WTA_NEW_STRING_1")
	i2 := table.AppendString("WTA_NEW_STRING_2")
	// Add duplicate of existing to check idempotency.
	i3 := table.AppendString("WTA_NEW_STRING_1")

	t.Logf("i1=%d i2=%d i3=%d", i1, i2, i3)
	if i1 == i2 {
		t.Error("i1 and i2 should differ")
	}
	if i3 != i1 {
		t.Errorf("i3 should equal i1 (dedup), got %d vs %d", i3, i1)
	}

	// Roundtrip.
	encoded := WriteTable(table)
	parsed, err := ParseTable(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.GlobalPool.Strings) != before+2 {
		t.Errorf("pool after: got %d, want %d", len(parsed.GlobalPool.Strings), before+2)
	}
	if parsed.GlobalPool.Strings[i1] != "WTA_NEW_STRING_1" {
		t.Errorf("string[i1]: got %q", parsed.GlobalPool.Strings[i1])
	}
	if parsed.GlobalPool.Strings[i2] != "WTA_NEW_STRING_2" {
		t.Errorf("string[i2]: got %q", parsed.GlobalPool.Strings[i2])
	}
}
