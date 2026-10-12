package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Entry is one cached build result.
type Entry struct {
	Fingerprint Fingerprint `json:"fingerprint"`
	Output      string      `json:"output"`
	OutputHash  Fingerprint `json:"outputHash"`
	BuiltAt     time.Time   `json:"builtAt"`
	Duration    int64       `json:"durationNs"`
}

// Store persists entries as a JSON file.
type Store struct {
	Path    string           `json:"-"`
	Entries map[string]Entry `json:"entries"`
}

// DefaultStore returns the user's cache file path.
func DefaultStore() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".wta-cache", "cache.json")
}

// Open loads an existing store or creates a new one.
func Open(path string) (*Store, error) {
	if path == "" {
		path = DefaultStore()
	}
	s := &Store{Path: path, Entries: map[string]Entry{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return &Store{Path: path, Entries: map[string]Entry{}}, nil
	}
	s.Path = path
	if s.Entries == nil {
		s.Entries = map[string]Entry{}
	}
	return s, nil
}

// Save writes the store to disk atomically.
func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

// Get returns the entry for a key if the fingerprint matches and output exists.
func (s *Store) Get(key string, fp Fingerprint) (Entry, bool) {
	e, ok := s.Entries[key]
	if !ok {
		return Entry{}, false
	}
	if e.Fingerprint != fp {
		return Entry{}, false
	}
	if _, err := os.Stat(e.Output); err != nil {
		return Entry{}, false
	}
	return e, true
}

// Put stores a new entry.
func (s *Store) Put(key string, e Entry) {
	s.Entries[key] = e
}

// Invalidate removes an entry.
func (s *Store) Invalidate(key string) {
	delete(s.Entries, key)
}

// Prune removes entries whose outputs no longer exist.
func (s *Store) Prune() int {
	n := 0
	for k, e := range s.Entries {
		if _, err := os.Stat(e.Output); err != nil {
			delete(s.Entries, k)
			n++
		}
	}
	return n
}

// Clear wipes all entries.
func (s *Store) Clear() {
	s.Entries = map[string]Entry{}
}
