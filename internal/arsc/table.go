package arsc

import (
	"encoding/binary"
	"fmt"
)

// Table is a fully parsed resources.arsc.
type Table struct {
	Raw        []byte
	GlobalPool *StringPool
	Packages   []*Package
}

// Package is a fully parsed TablePackage with its types.
type Package struct {
	Header      ChunkHeader
	ID          uint32
	Name        string
	TypeStrings *StringPool
	KeyStrings  *StringPool
	Types       map[uint8][]*TableType
	TypeSpecs   map[uint8]*TableTypeSpec
}

func ParseTable(buf []byte) (*Table, error) {
	hdr, err := ReadHeader(buf, 0)
	if err != nil {
		return nil, err
	}
	if hdr.Type != ChunkTable {
		return nil, fmt.Errorf("arsc: expected Table, got 0x%04X", hdr.Type)
	}
	t := &Table{Raw: buf}
	off := int(hdr.HeaderSize)

	for off+8 <= len(buf) {
		ch, err := ReadHeader(buf, off)
		if err != nil {
			return nil, err
		}
		if ch.Size == 0 || off+int(ch.Size) > len(buf) {
			break
		}
		switch ch.Type {
		case ChunkStringPool:
			if t.GlobalPool == nil {
				pool, err := ParseStringPool(buf, off)
				if err != nil {
					return nil, err
				}
				t.GlobalPool = pool
			}
		case ChunkTablePackage:
			pkg, err := parseFullPackage(buf, off)
			if err != nil {
				return nil, err
			}
			t.Packages = append(t.Packages, pkg)
		}
		off += int(ch.Size)
	}
	return t, nil
}

func parseFullPackage(buf []byte, off int) (*Package, error) {
	hdr, err := ReadHeader(buf, off)
	if err != nil {
		return nil, err
	}
	body := buf[off : off+int(hdr.Size)]
	if len(body) < 284 {
		return nil, fmt.Errorf("arsc: package too small")
	}
	pkg := &Package{
		Header:    hdr,
		ID:        binary.LittleEndian.Uint32(body[8:]),
		Types:     map[uint8][]*TableType{},
		TypeSpecs: map[uint8]*TableTypeSpec{},
	}
	pkg.Name = utf16LEString(body[12 : 12+256])

	typeStringsOff := binary.LittleEndian.Uint32(body[268:])
	keyStringsOff := binary.LittleEndian.Uint32(body[276:])
	baseOff := uint32(off)

	end := off + int(hdr.Size)
	inner := off + int(hdr.HeaderSize)

	for inner+8 <= end {
		ich, err := ReadHeader(buf, inner)
		if err != nil {
			break
		}
		if ich.Size == 0 || inner+int(ich.Size) > end {
			break
		}
		switch ich.Type {
		case ChunkStringPool:
			pool, err := ParseStringPool(buf, inner)
			if err == nil {
				rel := uint32(inner) - baseOff
				if rel == typeStringsOff {
					pkg.TypeStrings = pool
				} else if rel == keyStringsOff {
					pkg.KeyStrings = pool
				}
			}
		case ChunkTableTypeSpec:
			ts, err := ParseTypeSpec(buf, inner)
			if err == nil {
				pkg.TypeSpecs[ts.TypeID] = ts
			}
		case ChunkTableType:
			tt, err := ParseType(buf, inner)
			if err == nil {
				pkg.Types[tt.TypeID] = append(pkg.Types[tt.TypeID], tt)
			}
		}
		inner += int(ich.Size)
	}
	return pkg, nil
}

// ResolveString resolves a string reference into the global pool.
func (t *Table) ResolveString(idx uint32) string {
	if t.GlobalPool == nil {
		return ""
	}
	return t.GlobalPool.At(idx)
}

// ResolveEntry resolves a simple entry to a string.
func (t *Table) ResolveEntry(e *Entry) string {
	if e == nil || e.Value == nil {
		return ""
	}
	if e.Value.DataType == TypeString {
		return t.ResolveString(e.Value.Data)
	}
	if e.Value.DataType == TypeReference {
		ref := ResTableRef(e.Value.Data)
		return fmt.Sprintf("@0x%02X/%02X/%04X", ref.Package(), ref.Type(), ref.Entry())
	}
	return ""
}

// FindEntry finds a resource by package ID, type ID, entry index.
func (t *Table) FindEntry(pkgID uint32, typeID uint8, entryIdx uint32) *Entry {
	for _, pkg := range t.Packages {
		if pkg.ID != pkgID {
			continue
		}
		types := pkg.Types[typeID]
		if len(types) == 0 {
			return nil
		}
		var tt *TableType
		for _, candidate := range types {
			if candidate.Config.IsDefault() {
				tt = candidate
				break
			}
		}
		if tt == nil {
			tt = types[0]
		}
		if int(entryIdx) >= len(tt.Offsets) {
			return nil
		}
		entryOff := tt.Offsets[entryIdx]
		if entryOff == 0xFFFFFFFF {
			return nil
		}
		ent, err := ParseEntry(tt.Raw, int(tt.EntriesStart), entryOff)
		if err != nil {
			return nil
		}
		return ent
	}
	return nil
}

// SetString replaces the string at the given index in the global pool.
// Returns false if the index is out of range.
func (t *Table) SetString(idx uint32, value string) bool {
	if t.GlobalPool == nil || int(idx) >= len(t.GlobalPool.Strings) {
		return false
	}
	t.GlobalPool.Strings[idx] = value
	return true
}

// FindStringIndex returns the index of a string in the global pool, or -1.
func (t *Table) FindStringIndex(value string) int {
	if t.GlobalPool == nil {
		return -1
	}
	for i, s := range t.GlobalPool.Strings {
		if s == value {
			return i
		}
	}
	return -1
}

// SetEntryString replaces the value of a simple string entry.
// Returns false if the entry is complex or not a string.
func (t *Table) SetEntryString(pkgID uint32, typeID uint8, entryIdx uint32, newString string) bool {
	idx := t.FindStringIndex(newString)
	if idx < 0 {
		// Not found — append to pool and use new index.
		if t.GlobalPool == nil {
			return false
		}
		t.GlobalPool.Strings = append(t.GlobalPool.Strings, newString)
		idx = len(t.GlobalPool.Strings) - 1
	}

	for _, pkg := range t.Packages {
		if pkg.ID != pkgID {
			continue
		}
		types := pkg.Types[typeID]
		if len(types) == 0 {
			return false
		}
		// Prefer default config, otherwise use the first available one.
		var tt *TableType
		for _, candidate := range types {
			if candidate.Config.IsDefault() {
				tt = candidate
				break
			}
		}
		if tt == nil {
			tt = types[0]
		}
		{
			if int(entryIdx) >= len(tt.Offsets) {
				return false
			}
			entryOff := tt.Offsets[entryIdx]
			if entryOff == 0xFFFFFFFF {
				return false
			}
			// Modify in Raw.
			absOff := int(tt.EntriesStart) + int(entryOff)
			if absOff+16 > len(tt.Raw) {
				return false
			}
			// Data type at +11, data value at +12.
			tt.Raw[absOff+11] = TypeString
			binary.LittleEndian.PutUint32(tt.Raw[absOff+12:], uint32(idx))
			return true
		}
	}
	return false
}
