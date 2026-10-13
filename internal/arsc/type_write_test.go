package arsc

import (
	"archive/zip"
	"io"
	"testing"
)

func TestTypeRoundtrip(t *testing.T) {
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

	// Find first type.
	var original *TableType
	for _, list := range table.Packages[0].Types {
		if len(list) > 0 {
			original = list[0]
			break
		}
	}
	if original == nil {
		t.Skip("no types")
	}
	t.Logf("original type: id=0x%02X entries=%d", original.TypeID, original.EntryCount)

	// Serialize.
	encoded := WriteType(original)
	t.Logf("encoded: %d bytes (original raw: %d bytes)", len(encoded), len(original.Raw))

	// Parse back.
	parsed, err := ParseType(encoded, 0)
	if err != nil {
		t.Fatalf("ParseType: %v", err)
	}
	if parsed.TypeID != original.TypeID {
		t.Errorf("TypeID: got 0x%02X, want 0x%02X", parsed.TypeID, original.TypeID)
	}
	if parsed.EntryCount != original.EntryCount {
		t.Errorf("EntryCount: got %d, want %d", parsed.EntryCount, original.EntryCount)
	}
	if len(parsed.Offsets) != len(original.Offsets) {
		t.Errorf("Offsets count: got %d, want %d", len(parsed.Offsets), len(original.Offsets))
	}

	// Compare first few offsets.
	for i := 0; i < 5 && i < len(parsed.Offsets); i++ {
		if parsed.Offsets[i] != original.Offsets[i] {
			t.Errorf("offset[%d]: got 0x%X, want 0x%X", i, parsed.Offsets[i], original.Offsets[i])
		}
	}
}

func TestTypeSpecRoundtrip(t *testing.T) {
	orig := &TableTypeSpec{
		TypeID:     0x0F,
		EntryCount: 10,
	}
	data := WriteTypeSpec(orig)
	parsed, err := ParseTypeSpec(data, 0)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.TypeID != 0x0F {
		t.Errorf("TypeID: got 0x%02X", parsed.TypeID)
	}
	if parsed.EntryCount != 10 {
		t.Errorf("EntryCount: got %d", parsed.EntryCount)
	}
}
