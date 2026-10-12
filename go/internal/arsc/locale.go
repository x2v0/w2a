package arsc

import "encoding/binary"

// NormalizeLocale clears the Locale field in every Type's ResTableConfig.
// This makes the resources.arsc "locale-agnostic" so Android falls back to
// default (English/base) strings when this file is used.
// Returns the number of configs modified.
func (t *Table) NormalizeLocale() int {
	count := 0
	for _, pkg := range t.Packages {
		for _, list := range pkg.Types {
			for _, tt := range list {
				if tt.Config == nil || tt.Config.Locale == 0 {
					continue
				}
				tt.Config.Locale = 0
				tt.Config.Raw[8] = 0
				tt.Config.Raw[9] = 0
				tt.Config.Raw[10] = 0
				tt.Config.Raw[11] = 0
				// Also zero the SDK-related fields that might mark locale.
				binary.LittleEndian.PutUint32(tt.Config.Raw[8:], 0)
				count++
			}
		}
	}
	return count
}

// Locales returns a sorted list of distinct locale tags found in the table.
func (t *Table) Locales() []string {
	seen := map[string]struct{}{}
	for _, pkg := range t.Packages {
		for _, list := range pkg.Types {
			for _, tt := range list {
				if tt.Config == nil {
					continue
				}
				tag := tt.Config.Language()
				if cc := tt.Config.Country(); cc != "" {
					tag += "-" + cc
				}
				if tag == "" {
					tag = "default"
				}
				seen[tag] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}
