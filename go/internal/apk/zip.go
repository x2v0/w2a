package apk

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"time"
)

// ZipEntry holds one file inside an APK/ZIP archive.
type ZipEntry struct {
	Name     string
	Data     []byte
	Method   uint16 // zip.Store or zip.Deflate
	Modified uint32 // MS-DOS date/time
	ExternalAttr uint32
}

// ZipArchive is an in-memory representation of an APK.
type ZipArchive struct {
	Entries []*ZipEntry
	index   map[string]int
}

// LoadZip reads an APK/ZIP from bytes into memory.
func LoadZip(data []byte) (*ZipArchive, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("zip: %w", err)
	}
	a := &ZipArchive{index: map[string]int{}}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("zip: open %s: %w", f.Name, err)
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("zip: read %s: %w", f.Name, err)
		}
		e := &ZipEntry{
			Name:         f.Name,
			Data:         raw,
			Method:       f.Method,
			Modified:     uint32(f.ModifiedDate)<<16 | uint32(f.ModifiedTime),
			ExternalAttr: f.ExternalAttrs,
		}
		a.index[e.Name] = len(a.Entries)
		a.Entries = append(a.Entries, e)
	}
	return a, nil
}

// Get returns the entry with the given name, or nil.
func (a *ZipArchive) Get(name string) *ZipEntry {
	if idx, ok := a.index[name]; ok {
		return a.Entries[idx]
	}
	return nil
}

// Has reports whether the archive contains a given entry.
func (a *ZipArchive) Has(name string) bool {
	_, ok := a.index[name]
	return ok
}

// Replace inserts or overwrites an entry with STORED method.
// If the entry exists, its data is replaced. Method is forced to Store
// because AndroidManifest.xml and resources.arsc MUST be uncompressed.
func (a *ZipArchive) Replace(name string, data []byte) {
	if idx, ok := a.index[name]; ok {
		a.Entries[idx].Data = data
		a.Entries[idx].Method = zip.Store
		return
	}
	e := &ZipEntry{Name: name, Data: data, Method: zip.Store}
	a.index[name] = len(a.Entries)
	a.Entries = append(a.Entries, e)
}

// ReplaceDeflated inserts or overwrites an entry with DEFLATE method.
func (a *ZipArchive) ReplaceDeflated(name string, data []byte) {
	if idx, ok := a.index[name]; ok {
		a.Entries[idx].Data = data
		a.Entries[idx].Method = zip.Deflate
		return
	}
	e := &ZipEntry{Name: name, Data: data, Method: zip.Deflate}
	a.index[name] = len(a.Entries)
	a.Entries = append(a.Entries, e)
}

// Remove deletes an entry by name. No-op if not present.
func (a *ZipArchive) Remove(name string) {
	idx, ok := a.index[name]
	if !ok {
		return
	}
	a.Entries = append(a.Entries[:idx], a.Entries[idx+1:]...)
	delete(a.index, name)
	// Rebuild index.
	a.index = map[string]int{}
	for i, e := range a.Entries {
		a.index[e.Name] = i
	}
}

// Save serialises the archive back to bytes.
func (a *ZipArchive) Save() ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for _, e := range a.Entries {
		hdr := &zip.FileHeader{
			Name:   e.Name,
			Method: e.Method,
		}
		// Always use current time — original APKs often encode 1980/1981
		// (apksigner "meaningless date"), which looks broken in unzip.
		hdr.SetModTime(time.Now())
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return nil, fmt.Errorf("zip save: create %s: %w", e.Name, err)
		}
		if _, err := w.Write(e.Data); err != nil {
			return nil, fmt.Errorf("zip save: write %s: %w", e.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
