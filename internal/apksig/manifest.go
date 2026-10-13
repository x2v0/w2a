package apksig

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// ManifestFile represents the entries in META-INF/MANIFEST.MF.
type ManifestFile struct {
	Entries []ManifestEntry
}

// ManifestEntry holds one section of the manifest.
type ManifestEntry struct {
	Name string
	// SHA256 digest of the entry's contents, base64-encoded without padding.
	SHA256B64 string
}

// AddEntry computes SHA-256 of data and appends the entry.
func (m *ManifestFile) AddEntry(name string, data []byte) {
	sum := sha256.Sum256(data)
	b64 := base64StdNoPad(sum[:])
	m.Entries = append(m.Entries, ManifestEntry{Name: name, SHA256B64: b64})
}

// AddRaw appends an entry with a precomputed base64 SHA-256.
func (m *ManifestFile) AddRaw(name, sha256b64 string) {
	m.Entries = append(m.Entries, ManifestEntry{Name: name, SHA256B64: sha256b64})
}

// Encode serialises the manifest as text (per JAR spec).
// Sections are sorted by name for reproducibility.
func (m *ManifestFile) Encode() []byte {
	entries := append([]ManifestEntry(nil), m.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	var buf bytes.Buffer
	buf.WriteString("Manifest-Version: 1.0\r\n")
	buf.WriteString("Created-By: wta-buildapk (Go)\r\n")
	buf.WriteString("\r\n")

	for _, e := range entries {
		buf.WriteString("Name: " + e.Name + "\r\n")
		buf.WriteString("SHA-256-Digest: " + e.SHA256B64 + "\r\n")
		buf.WriteString("\r\n")
	}
	return buf.Bytes()
}

// SectionOf returns the raw bytes for one manifest section (used in SF).
func (m *ManifestFile) SectionOf(name string) []byte {
	for _, e := range m.Entries {
		if e.Name == name {
			var b bytes.Buffer
			b.WriteString("Name: " + e.Name + "\r\n")
			b.WriteString("SHA-256-Digest: " + e.SHA256B64 + "\r\n")
			b.WriteString("\r\n")
			return b.Bytes()
		}
	}
	return nil
}

// base64StdNoPad encodes to standard base64 without trailing '='.
func base64StdNoPad(data []byte) string {
	const tbl = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var sb strings.Builder
	n := len(data)
	for i := 0; i < n; i += 3 {
		var b0, b1, b2 byte
		b0 = data[i]
		if i+1 < n {
			b1 = data[i+1]
		}
		if i+2 < n {
			b2 = data[i+2]
		}
		sb.WriteByte(tbl[b0>>2])
		sb.WriteByte(tbl[((b0&0x03)<<4)|(b1>>4)])
		if i+1 < n {
			sb.WriteByte(tbl[((b1&0x0F)<<2)|(b2>>6)])
		}
		if i+2 < n {
			sb.WriteByte(tbl[b2&0x3F])
		}
	}
	return sb.String()
}

var _ = fmt.Sprintf
