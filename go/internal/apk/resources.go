package apk

import (
	"fmt"

	"github.com/x2v0/w2a/go/internal/arsc"
)

// LoadResources parses the resources.arsc inside the loaded APK.
// Returns nil if the APK has no resources table.
func (a *APK) LoadResources() (*arsc.Table, error) {
	entry := a.Archive.Get("resources.arsc")
	if entry == nil {
		return nil, fmt.Errorf("apk: no resources.arsc")
	}
	return arsc.ParseTable(entry.Data)
}

// SaveResources serialises the table back into the archive.
// Resources.arsc must be STORED (uncompressed).
func (a *APK) SaveResources(t *arsc.Table) error {
	if t == nil {
		return fmt.Errorf("apk: nil table")
	}
	data := arsc.WriteTable(t)
	a.Archive.Replace("resources.arsc", data)
	return nil
}
