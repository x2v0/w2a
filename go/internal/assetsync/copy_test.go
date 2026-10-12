package assetsync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "sub", "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestCopyMatching(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	_ = os.MkdirAll(src, 0o755)
	for _, n := range []string{"a.zip", "b.zip", "c.txt"} {
		_ = os.WriteFile(filepath.Join(src, n), []byte(n), 0o644)
	}
	n, err := CopyMatching(filepath.Join(src, "*.zip"), dst)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("expected 2, got %d", n)
	}
	for _, name := range []string{"a.zip", "b.zip"} {
		if _, err := os.Stat(filepath.Join(dst, name)); err != nil {
			t.Errorf("missing %s", name)
		}
	}
}
