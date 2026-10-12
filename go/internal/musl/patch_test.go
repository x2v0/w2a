package musl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyPatchSimple(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "src")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(sub, "file.txt")
	if err := os.WriteFile(target, []byte("line1\nline2\nline3\nline4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	patch := `--- a/src/file.txt
+++ b/src/file.txt
@@ -2,3 +2,4 @@
 line2
 line3
+inserted
 line4
`
	if err := ApplyPatch(dir, []byte(patch), 1); err != nil {
		t.Fatalf("ApplyPatch: %v", err)
	}

	got, _ := os.ReadFile(target)
	want := "line1\nline2\nline3\ninserted\nline4\n"
	if string(got) != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestApplyPatchRemoveLine(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "a")
	_ = os.MkdirAll(sub, 0o755)
	target := filepath.Join(sub, "f.txt")
	_ = os.WriteFile(target, []byte("keep\nremove\nkeep2\n"), 0o644)

	patch := `--- a/a/f.txt
+++ b/a/f.txt
@@ -1,3 +1,2 @@
 keep
-remove
 keep2
`
	if err := ApplyPatch(dir, []byte(patch), 1); err != nil {
		t.Fatalf("ApplyPatch: %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "keep\nkeep2\n" {
		t.Errorf("got %q", got)
	}
}

func TestStripPath(t *testing.T) {
	cases := []struct{ in, out string; n int }{
		{"a/src/file.txt", "src/file.txt", 1},
		{"a/b/c/d.txt", "c/d.txt", 2},
		{"file.txt", "file.txt", 0},
		{"a/file.txt", "a/file.txt", 0},
	}
	for _, c := range cases {
		if got := stripPath(c.in, c.n); got != c.out {
			t.Errorf("stripPath(%q, %d) = %q, want %q", c.in, c.n, got, c.out)
		}
	}
}
