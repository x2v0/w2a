package arsc

import (
	"encoding/binary"
	"fmt"
)

// AddStringEntry appends a new simple string entry to the given type
// (using the default config), updating offsets and entry count.
// Returns the new entry index, or -1 on failure.
func (t *Table) AddStringEntry(pkgID uint32, typeID uint8, value string) int {
	strIdx := t.AppendString(value)

	for _, pkg := range t.Packages {
		if pkg.ID != pkgID {
			continue
		}
		types := pkg.Types[typeID]
		if len(types) == 0 {
			return -1
		}
		// Prefer default config.
		var tt *TableType
		for _, c := range types {
			if c.Config.IsDefault() {
				tt = c
				break
			}
		}
		if tt == nil {
			tt = types[0]
		}
		return appendEntryToType(tt, strIdx)
	}
	return -1
}

// appendEntryToType serialises a new simple entry at the end of the entries
// block and records its offset.
func appendEntryToType(tt *TableType, strIdx uint32) int {
	// 16 bytes: size(2) flags(2) key(4) res_value(8)
	entry := make([]byte, 16)
	binary.LittleEndian.PutUint16(entry[0:], 8)  // value size
	binary.LittleEndian.PutUint16(entry[2:], 0)  // flags
	binary.LittleEndian.PutUint32(entry[4:], 0)  // key (unused)
	binary.LittleEndian.PutUint16(entry[8:], 8)  // res_value.size
	entry[10] = 0                                // res0
	entry[11] = TypeString                       // dataType
	binary.LittleEndian.PutUint32(entry[12:], strIdx)

	// Where does the entries block currently end?
	entriesBlockEnd := int(tt.EntriesStart)
	for i := 0; i < len(tt.Offsets); i++ {
		if tt.Offsets[i] == 0xFFFFFFFF {
			continue
		}
		end := int(tt.EntriesStart) + int(tt.Offsets[i]) + entrySizeAt(tt.Raw, int(tt.EntriesStart)+int(tt.Offsets[i]))
		if end > entriesBlockEnd {
			entriesBlockEnd = end
		}
	}
	if entriesBlockEnd < int(tt.EntriesStart) {
		entriesBlockEnd = int(tt.EntriesStart)
	}

	// Append new entry bytes to Raw at entriesBlockEnd.
	newRaw := append(tt.Raw[:entriesBlockEnd], entry...)
	tt.Raw = newRaw

	// Compute offset of new entry.
	newOffset := uint32(entriesBlockEnd - int(tt.EntriesStart))

	// Append to Offsets.
	tt.Offsets = append(tt.Offsets, newOffset)
	tt.EntryCount = uint32(len(tt.Offsets))

	return len(tt.Offsets) - 1
}

// entrySizeAt returns the size in bytes of an entry starting at absOff.
// For simple entries: 16 bytes (8 header + 8 value).
// For bags: 16 + 12*count.
func entrySizeAt(raw []byte, absOff int) int {
	if absOff+4 > len(raw) {
		return 16
	}
	flags := binary.LittleEndian.Uint16(raw[absOff+2:])
	if flags&EntryFlagComplex == 0 {
		return 16
	}
	if absOff+16 > len(raw) {
		return 16
	}
	count := binary.LittleEndian.Uint32(raw[absOff+12:])
	return 16 + int(count)*12
}

// Ensure imports are used
var _ = fmt.Sprintf
