package arsc

// IsArabic returns true if s contains any Arabic-range characters.
func IsArabic(s string) bool {
	for _, r := range s {
		switch {
		case r >= 0x0600 && r <= 0x06FF, // Arabic
			r >= 0x0750 && r <= 0x077F, // Arabic Supplement
			r >= 0x08A0 && r <= 0x08FF, // Arabic Extended-A
			r >= 0xFB50 && r <= 0xFDFF, // Presentation Forms-A
			r >= 0xFE70 && r <= 0xFEFF: // Presentation Forms-B
			return true
		}
	}
	return false
}

// StripArabic replaces every string containing Arabic characters with repl.
// Returns the number of modified strings.
func (t *Table) StripArabic(repl string) int {
	if t.GlobalPool == nil {
		return 0
	}
	count := 0
	for i, s := range t.GlobalPool.Strings {
		if IsArabic(s) {
			t.GlobalPool.Strings[i] = repl
			count++
		}
	}
	return count
}
