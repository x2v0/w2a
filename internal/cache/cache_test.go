package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFilesStable(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("hello"), 0o644)
	os.WriteFile(b, []byte("world"), 0o644)

	fp1, err := HashFiles([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := HashFiles([]string{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if fp1 != fp2 {
		t.Errorf("order should not matter: %s vs %s", fp1, fp2)
	}

	// Modify content.
	os.WriteFile(a, []byte("HELLO"), 0o644)
	fp3, _ := HashFiles([]string{a, b})
	if fp1 == fp3 {
		t.Error("fingerprint should change after edit")
	}
}

func TestStoreRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "out.apk")
	os.WriteFile(out, []byte("fake apk"), 0o644)

	s.Put("build1", Entry{
		Fingerprint: "abc123",
		Output:      out,
		OutputHash:  "def456",
	})
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	// Reopen.
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := s2.Get("build1", "abc123")
	if !ok {
		t.Fatal("entry should be found")
	}
	if e.Output != out {
		t.Errorf("wrong output: %q", e.Output)
	}

	// Mismatched fingerprint -> miss.
	_, ok = s2.Get("build1", "wrong")
	if ok {
		t.Error("should miss on fingerprint mismatch")
	}
}

func TestStorePrune(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Path: filepath.Join(dir, "c.json"), Entries: map[string]Entry{}}
	s.Put("gone", Entry{Output: filepath.Join(dir, "missing.apk")})
	s.Put("live", Entry{Output: dir}) // dir exists

	n := s.Prune()
	if n != 1 {
		t.Errorf("expected 1 pruned, got %d", n)
	}
	if _, ok := s.Entries["gone"]; ok {
		t.Error("gone should be removed")
	}
	if _, ok := s.Entries["live"]; !ok {
		t.Error("live should stay")
	}
}
