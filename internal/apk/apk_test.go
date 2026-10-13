package apk

import (
	"bytes"
	"testing"
)

func TestZipRoundtrip(t *testing.T) {
	a := &ZipArchive{index: map[string]int{}}
	a.Replace("AndroidManifest.xml", []byte("dummy-manifest"))
	a.ReplaceDeflated("classes.dex", bytes.Repeat([]byte{0xAB}, 1000))
	a.Replace("resources.arsc", []byte("dummy-arsc"))
	a.ReplaceDeflated("assets/data.txt", []byte("hello world"))

	saved, err := a.Save()
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Logf("archive size: %d bytes", len(saved))

	loaded, err := LoadZip(saved)
	if err != nil {
		t.Fatalf("LoadZip: %v", err)
	}

	for _, name := range []string{"AndroidManifest.xml", "classes.dex", "resources.arsc", "assets/data.txt"} {
		e := loaded.Get(name)
		if e == nil {
			t.Errorf("missing entry: %s", name)
			continue
		}
		t.Logf("%s: %d bytes, method=%d", name, len(e.Data), e.Method)
	}

	m := loaded.Get("AndroidManifest.xml")
	if m.Method != 0 {
		t.Errorf("manifest should be STORED, got method=%d", m.Method)
	}
	data := loaded.Get("assets/data.txt")
	if string(data.Data) != "hello world" {
		t.Errorf("data mismatch: %q", string(data.Data))
	}
}

func TestZipRemoveAndReplace(t *testing.T) {
	a := &ZipArchive{index: map[string]int{}}
	a.Replace("a.txt", []byte("A"))
	a.Replace("b.txt", []byte("B"))
	a.Replace("c.txt", []byte("C"))

	a.Remove("b.txt")
	if a.Has("b.txt") {
		t.Error("b.txt should be removed")
	}
	a.Replace("a.txt", []byte("A2"))
	if string(a.Get("a.txt").Data) != "A2" {
		t.Error("a.txt should be replaced")
	}
}
