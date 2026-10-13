package modules

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// LoadJSON reads and parses a JSON file. On any failure it records a
// diagnostic in the report and returns nil.
func LoadJSON(r *Report, path, where string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			r.Error(where, fmt.Sprintf("file not found: %s", path))
		} else {
			r.Error(where, fmt.Sprintf("read error: %v", err))
		}
		return nil
	}
	if !utf8.Valid(data) {
		r.Error(where, "file must be UTF-8")
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		r.Error(where, fmt.Sprintf("invalid JSON: %v", err))
		return nil
	}
	return out
}

// ReadJSONFile is a lenient helper used by callers that don't want
// diagnostics (e.g. reading the previous submissions output).
func ReadJSONFile(path string) (map[string]any, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, false
	}
	return out, true
}

// ReadExistingSubmissions returns the previous run's `submissions` map.
func ReadExistingSubmissions(path string) map[string]any {
	if m, ok := ReadJSONFile(path); ok {
		if subs, ok := m["submissions"].(map[string]any); ok {
			return subs
		}
	}
	return map[string]any{}
}

// ReadRegistry returns the registry's `modules` array, ignoring bad entries.
func ReadRegistry(path string) []map[string]any {
	if m, ok := ReadJSONFile(path); ok {
		if arr, ok := m["modules"].([]any); ok {
			out := make([]map[string]any, 0, len(arr))
			for _, item := range arr {
				if entry, ok := item.(map[string]any); ok {
					out = append(out, entry)
				}
			}
			return out
		}
	}
	return nil
}

// AbsOrPath returns an absolute path (best effort) — used for error messages.
func AbsOrPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
