package arsc

import (
	"encoding/binary"
	"sort"
	"strings"
)

// LocaleInfo describes one distinct locale present in the table.
type LocaleInfo struct {
	Tag       string // "ar", "ru-en", "default"
	TypeCount int    // how many Type chunks carry this locale
}

// ListLocales returns all distinct locales in the table.
func (t *Table) ListLocales() []LocaleInfo {
	counts := map[string]int{}
	for _, pkg := range t.Packages {
		for _, list := range pkg.Types {
			for _, tt := range list {
				tag := "default"
				if tt.Config != nil {
					lang := tt.Config.Language()
					cc := tt.Config.Country()
					switch {
					case lang == "":
						tag = "default"
					case cc != "":
						tag = lang + "-" + cc
					default:
						tag = lang
					}
				}
				counts[tag]++
			}
		}
	}
	out := make([]LocaleInfo, 0, len(counts))
	for k, v := range counts {
		out = append(out, LocaleInfo{Tag: k, TypeCount: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	return out
}

// StripLocale zeroes the locale of every Type whose config matches `tag`.
// tag may be "ar", "ru-en", or "*" for all. Returns count modified.
func (t *Table) StripLocale(tag string) int {
	matchAll := tag == "*" || tag == ""
	wantLang, wantCC := splitTag(tag)
	count := 0
	for _, pkg := range t.Packages {
		for _, list := range pkg.Types {
			for _, tt := range list {
				if tt.Config == nil {
					continue
				}
				lang := tt.Config.Language()
				cc := tt.Config.Country()
				curTag := lang
				if cc != "" {
					curTag += "-" + cc
				}
				if curTag == "" {
					curTag = "default"
				}
				if matchAll || curTag == tag ||
					(wantLang != "" && lang == wantLang && wantCC == "") {
					// Zero the Locale field (4 bytes: language + country).
					if len(tt.Config.Raw) >= 12 {
						binary.LittleEndian.PutUint32(tt.Config.Raw[8:], 0)
						tt.Config.Locale = 0
						count++
					}
				}
			}
		}
	}
	return count
}

// KeepLocales zeroes locales NOT in the whitelist.
func (t *Table) KeepLocales(keep []string) int {
	keepSet := map[string]struct{}{}
	for _, k := range keep {
		keepSet[strings.ToLower(k)] = struct{}{}
	}
	count := 0
	for _, pkg := range t.Packages {
		for _, list := range pkg.Types {
			for _, tt := range list {
				if tt.Config == nil {
					continue
				}
				lang := tt.Config.Language()
				cc := tt.Config.Country()
				curTag := lang
				if cc != "" {
					curTag += "-" + cc
				}
				if curTag == "" {
					continue // default — never stripped
				}
				if _, ok := keepSet[strings.ToLower(curTag)]; ok {
					continue
				}
				if _, ok := keepSet[strings.ToLower(lang)]; ok {
					continue
				}
				if len(tt.Config.Raw) >= 12 {
					binary.LittleEndian.PutUint32(tt.Config.Raw[8:], 0)
					tt.Config.Locale = 0
					count++
				}
			}
		}
	}
	return count
}

func splitTag(tag string) (lang, cc string) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if i := strings.IndexByte(tag, '-'); i >= 0 {
		return tag[:i], tag[i+1:]
	}
	return tag, ""
}
