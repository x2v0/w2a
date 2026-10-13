package modules

import "testing"

func TestReportRender(t *testing.T) {
	r := &Report{}
	if !r.OK() {
		t.Fatal("empty report should be OK")
	}
	if got := r.Render(); got != "\nAll module checks passed." {
		t.Errorf("got %q", got)
	}
	r.Error("where", "msg")
	if r.OK() {
		t.Fatal("report with error should not be OK")
	}
}

func TestKebabCase(t *testing.T) {
	valid := []string{"abc", "a-b-c", "a1-b2", "plugin-name"}
	invalid := []string{"ABC", "a_b", "-a", "a-", "a--b", ""}
	for _, s := range valid {
		if !KebabCaseRE.MatchString(s) {
			t.Errorf("kebab should match %q", s)
		}
	}
	for _, s := range invalid {
		if KebabCaseRE.MatchString(s) {
			t.Errorf("kebab should NOT match %q", s)
		}
	}
}

func TestSemver(t *testing.T) {
	valid := []string{"1", "1.2", "1.2.3", "1.2.3.4", "1.0.0-alpha", "2.0.0+build.1"}
	invalid := []string{"", "v1", "abc", "1.", ".1"}
	for _, s := range valid {
		if !SemverRE.MatchString(s) {
			t.Errorf("semver should match %q", s)
		}
	}
	for _, s := range invalid {
		if SemverRE.MatchString(s) {
			t.Errorf("semver should NOT match %q", s)
		}
	}
}

func TestStoreID(t *testing.T) {
	valid := "abcdefghijklmnopabcdefghijklmnop"
	if !StoreIDRE.MatchString(valid) {
		t.Error("valid store id should match")
	}
	if StoreIDRE.MatchString("short") {
		t.Error("short should not match")
	}
	if StoreIDRE.MatchString("zbcdefghijklmnopabcdefghijklmnop") {
		t.Error("z should not match (only a-p)")
	}
}

func TestExpectListOfStr(t *testing.T) {
	r := &Report{}
	got := ExpectListOfStr(r, "w", []any{"a", "b"}, "field")
	if len(got) != 2 || got[0] != "a" {
		t.Errorf("got %v", got)
	}
	r2 := &Report{}
	ExpectListOfStr(r2, "w", []any{"a", 123}, "field")
	if r2.OK() {
		t.Error("mixed list should error")
	}
}

func TestPRNumbersFromText(t *testing.T) {
	got := PRNumbersFromText("feat: add module (#123)", "fixup for #456 and (#123)")
	if len(got) != 2 {
		t.Fatalf("expected 2 unique PRs, got %v", got)
	}
	if got[0] != 123 || got[1] != 456 {
		t.Errorf("got %v", got)
	}
}
