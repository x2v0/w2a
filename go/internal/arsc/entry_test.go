package arsc

import (
	"archive/zip"
	"io"
	"testing"
)

func TestParseStringPoolAndEntries(t *testing.T) {
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

	// Global string pool at offset 12.
	pool, err := ParseStringPool(data, 12)
	if err != nil {
		t.Fatalf("ParseStringPool: %v", err)
	}
	t.Logf("global pool: isUTF8=%v count=%d", pool.IsUTF8, len(pool.Strings))
	for i := 0; i < 5 && i < len(pool.Strings); i++ {
		t.Logf("  [%d] %q", i, pool.Strings[i])
	}

	// Find package chunk, parse types.
	off := 12
	for off+8 <= len(data) {
		ch, _ := ReadHeader(data, off)
		if ch.Size == 0 || off+int(ch.Size) > len(data) {
			break
		}
		if ch.Type == ChunkTablePackage {
			end := off + int(ch.Size)
			inner := off + int(ch.HeaderSize)
			for inner+8 <= end {
				ich, _ := ReadHeader(data, inner)
				if ich.Size == 0 || inner+int(ich.Size) > end {
					break
				}
				if ich.Type == ChunkTableType {
					tt, err := ParseType(data, inner)
					if err != nil {
						t.Logf("  ParseType err: %v", err)
						inner += int(ich.Size)
						continue
					}
					t.Logf("Type id=0x%02X entries=%d config=%s",
						tt.TypeID, tt.EntryCount, tt.Config)
					// Parse first 3 non-empty entries.
					shown := 0
					for i, entryOff := range tt.Offsets {
						if entryOff == 0xFFFFFFFF {
							continue
						}
						ent, err := ParseEntry(tt.Raw, int(tt.EntriesStart), entryOff)
						if err != nil {
							t.Logf("    entry[%d] err: %v", i, err)
							continue
						}
						if ent == nil {
							continue
						}
						if ent.Value != nil {
							t.Logf("    entry[%d] type=0x%02X data=0x%08X",
								i, ent.Value.DataType, ent.Value.Data)
						} else if ent.Bag != nil {
							t.Logf("    entry[%d] BAG items=%d parent=0x%08X",
								i, len(ent.Bag.Items), ent.Bag.ParentID)
						}
						shown++
						if shown >= 3 {
							break
						}
					}
				}
				inner += int(ich.Size)
			}
			return
		}
		off += int(ch.Size)
	}
}
