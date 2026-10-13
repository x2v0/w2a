package arsc

import "strings"

// Script ranges for common Unicode blocks.
type scriptRange struct {
	name  string
	start rune
	end   rune
}

var scripts = []scriptRange{
	{"arabic", 0x0600, 0x06FF},
	{"arabic", 0x0750, 0x077F},
	{"arabic", 0x08A0, 0x08FF},
	{"arabic", 0xFB50, 0xFDFF},
	{"arabic", 0xFE70, 0xFEFF},
	{"cyrillic", 0x0400, 0x04FF},
	{"cyrillic", 0x0500, 0x052F},
	{"hebrew", 0x0590, 0x05FF},
	{"greek", 0x0370, 0x03FF},
	{"devanagari", 0x0900, 0x097F},
	{"thai", 0x0E00, 0x0E7F},
	{"cjk", 0x4E00, 0x9FFF},
	{"cjk", 0x3400, 0x4DBF},
	{"kana", 0x3040, 0x30FF},
	{"hangul", 0xAC00, 0xD7AF},
	{"hangul", 0x1100, 0x11FF},
}

// DetectScript returns the dominant script name of s, or "" if Latin/ASCII.
func DetectScript(s string) string {
	counts := map[string]int{}
	for _, r := range s {
		if r < 0x80 {
			continue // skip ASCII
		}
		for _, sr := range scripts {
			if r >= sr.start && r <= sr.end {
				counts[sr.name]++
				break
			}
		}
	}
	best, n := "", 0
	for k, v := range counts {
		if v > n {
			best, n = k, v
		}
	}
	return best
}

// StripScript replaces all strings whose dominant script matches `name`.
// Returns number of strings modified.
func (t *Table) StripScript(name, repl string) int {
	if t.GlobalPool == nil {
		return 0
	}
	name = strings.ToLower(name)
	count := 0
	for i, s := range t.GlobalPool.Strings {
		if DetectScript(s) == name {
			t.GlobalPool.Strings[i] = repl
			count++
		}
	}
	return count
}

// IsArabic — backward compat helper.
func IsArabic(s string) bool { return DetectScript(s) == "arabic" }

// StripArabic — backward compat helper.
func (t *Table) StripArabic(repl string) int { return t.StripScript("arabic", repl) }

// NormalizeLocale zeroes the Locale field in every Type's ResTableConfig.
// Returns the number of configs modified.
func (t *Table) NormalizeLocale() int {
	count := 0
	for _, pkg := range t.Packages {
		for _, list := range pkg.Types {
			for _, tt := range list {
				if tt.Config == nil || tt.Config.Locale == 0 {
					continue
				}
				if len(tt.Config.Raw) >= 12 {
					tt.Config.Locale = 0
					tt.Config.Raw[8] = 0
					tt.Config.Raw[9] = 0
					tt.Config.Raw[10] = 0
					tt.Config.Raw[11] = 0
					count++
				}
			}
		}
	}
	return count
}
