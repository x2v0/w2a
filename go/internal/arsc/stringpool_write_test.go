package arsc

import (
	"bytes"
	"testing"
)

func TestStringPoolRoundtripUTF8(t *testing.T) {
	original := &StringPool{
		IsUTF8: true,
		Strings: []string{
			"app_name",
			"Hello, World!",
			"التنقل إلى أعلى",
			"日本語テスト",
			"",
			"short",
		},
	}

	data := WriteStringPool(original)
	t.Logf("encoded: %d bytes", len(data))

	parsed, err := ParseStringPool(data, 0)
	if err != nil {
		t.Fatalf("ParseStringPool: %v", err)
	}
	if !parsed.IsUTF8 {
		t.Error("IsUTF8 mismatch")
	}
	if len(parsed.Strings) != len(original.Strings) {
		t.Fatalf("count: got %d, want %d", len(parsed.Strings), len(original.Strings))
	}
	for i, want := range original.Strings {
		if parsed.Strings[i] != want {
			t.Errorf("string[%d]: got %q, want %q", i, parsed.Strings[i], want)
		}
	}
}

func TestStringPoolRoundtripUTF16(t *testing.T) {
	original := &StringPool{
		IsUTF8:  false,
		Strings: []string{"simple", "日本語", "emoji: 🎉"},
	}
	data := WriteStringPool(original)
	parsed, err := ParseStringPool(data, 0)
	if err != nil {
		t.Fatalf("ParseStringPool: %v", err)
	}
	if parsed.IsUTF8 {
		t.Error("should be UTF-16")
	}
	for i, want := range original.Strings {
		if parsed.Strings[i] != want {
			t.Errorf("string[%d]: got %q, want %q", i, parsed.Strings[i], want)
		}
	}
}

func TestVarLenEncoding(t *testing.T) {
	cases := []struct {
		n       int
		wantLen int
	}{
		{0, 1}, {1, 1}, {127, 1}, {128, 2}, {255, 2}, {1000, 2}, {16383, 2},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		writeVarLenSP(&buf, c.n)
		if buf.Len() != c.wantLen {
			t.Errorf("n=%d: got %d bytes, want %d", c.n, buf.Len(), c.wantLen)
		}
	}
}
