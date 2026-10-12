package arsc

import (
	"encoding/binary"
	"fmt"
)

// TablePackage is a parsed package chunk.
type TablePackage struct {
	Header      ChunkHeader
	ID          uint32
	Name        string
	TypeStrings uint32 // index into global string pool of type names
	KeyStrings  uint32 // index into global string pool of key names
	Types       []*TableType
	TypeSpecs   []*TableTypeSpec
	// Raw retains the underlying bytes for this package.
	Raw []byte
}

// ParsePackage parses a TablePackage chunk at the given offset in buf.
func ParsePackage(buf []byte, off int) (*TablePackage, error) {
	hdr, err := ReadHeader(buf, off)
	if err != nil {
		return nil, err
	}
	if hdr.Type != ChunkTablePackage {
		return nil, fmt.Errorf("arsc: expected TablePackage (0x%04X), got 0x%04X",
			ChunkTablePackage, hdr.Type)
	}
	if off+int(hdr.Size) > len(buf) {
		return nil, fmt.Errorf("arsc: package size overflow")
	}

	body := buf[off : off+int(hdr.Size)]
	if len(body) < 8+4+256 {
		return nil, fmt.Errorf("arsc: package too small")
	}

	pkg := &TablePackage{
		Header: hdr,
		Raw:    body,
		ID:     binary.LittleEndian.Uint32(body[8:]),
	}

	// Package name: 256 UTF-16 code units (128 pairs), starting at offset 12.
	pkg.Name = utf16LEString(body[12 : 12+256])

	// Type/key string indices (both point into the global string pool).
	// Layout: 8 (header) + 4 (id) + 256 (name) + 4 (typeStrings) +
	//         4 (lastPublicType) + 4 (keyStrings) + 4 (lastPublicKey)
	if len(body) >= 12+256+16 {
		pkg.TypeStrings = binary.LittleEndian.Uint32(body[268:])
		pkg.KeyStrings = binary.LittleEndian.Uint32(body[276:])
	}

	return pkg, nil
}

// utf16LEString decodes a fixed-size UTF-16LE string (null-terminated).
func utf16LEString(buf []byte) string {
	var runes []rune
	for i := 0; i+1 < len(buf); i += 2 {
		ch := binary.LittleEndian.Uint16(buf[i:])
		if ch == 0 {
			break
		}
		runes = append(runes, rune(ch))
	}
	return string(runes)
}
