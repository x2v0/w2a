package arsc

import (
	"archive/zip"
	"encoding/binary"
	"io"
	"os"
	"testing"
)

func findAnyAPK() string {
	p := os.Getenv("WTA_TEST_APK")
	if p == "" {
		return ""
	}
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

func TestParseTableHeader(t *testing.T) {
	apkPath := findAnyAPK()
	if apkPath == "" {
		t.Skip("no test APK — set WTA_TEST_APK")
	}
	zr, err := zip.OpenReader(apkPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	var rc io.ReadCloser
	for _, f := range zr.File {
		if f.Name == "resources.arsc" {
			rc, err = f.Open()
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if rc == nil {
		t.Skip("no resources.arsc in APK")
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	t.Logf("resources.arsc: %d bytes", len(data))

	hdr, err := ReadHeader(data, 0)
	if err != nil {
		t.Fatal(err)
	}
	if hdr.Type != ChunkTable {
		t.Errorf("expected ChunkTable (0x%04X), got 0x%04X", ChunkTable, hdr.Type)
	}
	t.Logf("header: type=0x%04X headerSize=%d size=%d",
		hdr.Type, hdr.HeaderSize, hdr.Size)

	if len(data) < 12 {
		t.Fatal("too small")
	}
	pkgCount := binary.LittleEndian.Uint32(data[8:])
	t.Logf("packageCount: %d", pkgCount)
	if pkgCount == 0 {
		t.Error("no packages")
	}

	off := int(hdr.HeaderSize)
	for off+8 <= len(data) {
		ch, err := ReadHeader(data, off)
		if err != nil {
			t.Logf("chunk error at %d: %v", off, err)
			break
		}
		if ch.Size == 0 || off+int(ch.Size) > len(data) {
			t.Logf("chunk size overflow at off=%d", off)
			break
		}
		t.Logf("chunk at %d: type=0x%04X size=%d", off, ch.Type, ch.Size)
		off += int(ch.Size)
	}
}
