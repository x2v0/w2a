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

// SetAppName replaces the resource pointed to by AndroidManifest's
// android:label attribute with newName.
func (a *APK) SetAppName(newName string) error {
	if a.Manifest == nil || a.Manifest.Root == nil {
		return fmt.Errorf("apk: no manifest")
	}
	app := a.Manifest.Root.FindChild("application")
	if app == nil {
		return fmt.Errorf("apk: no <application> in manifest")
	}
	label := app.Attr("label")
	if label == nil {
		return fmt.Errorf("apk: no android:label on <application>")
	}

	table, err := a.LoadResources()
	if err != nil {
		return fmt.Errorf("apk: load resources: %w", err)
	}

	// If the label is a string reference, resolve and modify in the pool.
	if label.ValueType == 0x03 { // TypeString
		idx := label.ValueData
		if !table.SetString(idx, newName) {
			return fmt.Errorf("apk: string index %d out of range", idx)
		}
		return a.SaveResources(table)
	}

	// If it's a reference (0x01) to @string/app_name, resolve it.
	if label.ValueType == 0x01 { // TypeReference
		// Data format: 0xPPTTEEEE (package, type, entry).
		ref := label.ValueData
		pkgID := uint32(ref >> 24)
		typeID := uint8((ref >> 16) & 0xFF)
		entryID := uint32(ref & 0xFFFF)
		if !table.SetEntryString(pkgID, typeID, entryID, newName) {
			return fmt.Errorf("apk: cannot modify @0x%02X/%02X/%04X", pkgID, typeID, entryID)
		}
		return a.SaveResources(table)
	}

	return fmt.Errorf("apk: unsupported label value type 0x%02X", label.ValueType)
}

// SetStringResource replaces a string in the global pool by index.
// Useful when the caller knows the specific string index to edit.
func (a *APK) SetStringResource(idx uint32, value string) error {
	table, err := a.LoadResources()
	if err != nil {
		return err
	}
	if !table.SetString(idx, value) {
		return fmt.Errorf("apk: string index %d out of range", idx)
	}
	return a.SaveResources(table)
}

// ListStrings prints a subset of the string pool for debugging.
func (a *APK) ListStrings(offset, limit int) ([]string, error) {
	table, err := a.LoadResources()
	if err != nil {
		return nil, err
	}
	if table.GlobalPool == nil {
		return nil, fmt.Errorf("apk: no string pool")
	}
	strs := table.GlobalPool.Strings
	if offset >= len(strs) {
		return nil, nil
	}
	end := offset + limit
	if end > len(strs) {
		end = len(strs)
	}
	return strs[offset:end], nil
}

// StripArabic replaces all Arabic strings in resources.arsc with repl.
// Returns the number of modified strings.
func (a *APK) StripArabic(repl string) (int, error) {
	table, err := a.LoadResources()
	if err != nil {
		return 0, err
	}
	n := table.StripArabic(repl)
	if err := a.SaveResources(table); err != nil {
		return n, err
	}
	return n, nil
}

// NormalizeLocale strips the locale from all ResTableConfig entries.
// Returns number of configs modified.
func (a *APK) NormalizeLocale() (int, error) {
	table, err := a.LoadResources()
	if err != nil {
		return 0, err
	}
	n := table.NormalizeLocale()
	if err := a.SaveResources(table); err != nil {
		return n, err
	}
	return n, nil
}
