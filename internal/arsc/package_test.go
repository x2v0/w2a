package arsc

import (
	"archive/zip"
	"io"
	"testing"
)

func TestParsePackage(t *testing.T) {
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

	// Find first package chunk.
	off := 12
	for off+8 <= len(data) {
		ch, _ := ReadHeader(data, off)
		if ch.Size == 0 || off+int(ch.Size) > len(data) {
			break
		}
		if ch.Type == ChunkTablePackage {
			pkg, err := ParsePackage(data, off)
			if err != nil {
				t.Fatalf("ParsePackage: %v", err)
			}
			t.Logf("package: id=0x%02X name=%q", pkg.ID, pkg.Name)
			t.Logf("typeStrings=%d keyStrings=%d", pkg.TypeStrings, pkg.KeyStrings)

			// Walk inner chunks.
			inner := off + int(ch.HeaderSize)
			end := off + int(ch.Size)
			for inner+8 <= end {
				ich, _ := ReadHeader(data, inner)
				if ich.Size == 0 || inner+int(ich.Size) > end {
					break
				}
				switch ich.Type {
				case ChunkTableTypeSpec:
					ts, err := ParseTypeSpec(data, inner)
					if err == nil {
						t.Logf("  TypeSpec typeId=0x%02X entryCount=%d",
							ts.TypeID, ts.EntryCount)
					}
				case ChunkTableType:
					tt, err := ParseType(data, inner)
					if err == nil {
						t.Logf("  Type     typeId=0x%02X entryCount=%d config=%s",
							tt.TypeID, tt.EntryCount, tt.Config)
					}
				}
				inner += int(ich.Size)
			}
			return
		}
		off += int(ch.Size)
	}
	t.Fatal("no package chunk found")
}
