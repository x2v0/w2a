package modules

import "fmt"

// IsStr reports whether v is a JSON string.
func IsStr(v any) bool {
	_, ok := v.(string)
	return ok
}

// ExpectStr validates a nullable string field. Returns the string or "".
// nil is allowed (treated as absent).
func ExpectStr(r *Report, where string, v any, field string) string {
	if v == nil {
		return ""
	}
	if !IsStr(v) {
		r.Error(where, fmt.Sprintf("`%s` must be a string, got %T", field, v))
		return ""
	}
	return v.(string)
}

// ExpectListOfStr validates a nullable list-of-strings field.
// Returns the list or nil. Wrong type -> error, returns nil.
func ExpectListOfStr(r *Report, where string, v any, field string) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		r.Error(where, fmt.Sprintf("`%s` must be a list of strings", field))
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			r.Error(where, fmt.Sprintf("`%s` must be a list of strings", field))
			return nil
		}
		out = append(out, s)
	}
	return out
}

// ValidateURLMatches checks the optional urlMatches list-of-objects field.
func ValidateURLMatches(r *Report, where string, v any) {
	if v == nil {
		return
	}
	arr, ok := v.([]any)
	if !ok {
		r.Error(where, "`urlMatches` must be a list")
		return
	}
	for i, item := range arr {
		loc := fmt.Sprintf("%s::urlMatches[%d]", where, i)
		rule, ok := item.(map[string]any)
		if !ok {
			r.Error(loc, "must be an object with at least `pattern`")
			continue
		}
		pattern, ok := rule["pattern"].(string)
		if !ok || pattern == "" {
			r.Error(loc, "`pattern` is required and must be a non-empty string")
		}
		for _, flag := range []string{"isRegex", "exclude"} {
			if raw, present := rule[flag]; present {
				if _, isBool := raw.(bool); !isBool {
					r.Error(loc, fmt.Sprintf("`%s` must be a boolean", flag))
				}
			}
		}
	}
}

// ValidateAuthor checks the optional author object field.
func ValidateAuthor(r *Report, where string, v any) {
	if v == nil {
		return
	}
	author, ok := v.(map[string]any)
	if !ok {
		r.Error(where, "`author` must be an object")
		return
	}
	name, ok := author["name"].(string)
	if !ok || strings_TrimSpaceEmpty(name) {
		r.Error(where, "`author.name` is required")
	}
	for _, key := range []string{"email", "url", "qq"} {
		raw, present := author[key]
		if !present || raw == nil {
			continue
		}
		if _, isStr := raw.(string); !isStr {
			r.Error(where, fmt.Sprintf("`author.%s` must be a string", key))
		}
	}
}

// strings_TrimSpaceEmpty is a tiny local helper so we don't import strings
// in every file just to check trim-space emptiness.
func strings_TrimSpaceEmpty(s string) bool {
	for _, ch := range s {
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			return false
		}
	}
	return true
}
