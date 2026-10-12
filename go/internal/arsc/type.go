package arsc

import (
	"encoding/binary"
	"fmt"
)

// TableTypeSpec describes the number of entries for one type ID.
type TableTypeSpec struct {
	Header   ChunkHeader
	TypeID   uint8
	Res0     uint8
	Res1     uint16
	EntryCount uint32
	Configs  []*ResTableConfig // one entry per type chunk
}

// TableType holds the actual resource values for one type+config pair.
type TableType struct {
	Header      ChunkHeader
	TypeID      uint8
	Res0        uint8
	Res1        uint16
	EntryCount  uint32
	EntriesStart uint32
	Config      *ResTableConfig
	Offsets     []uint32 // entry offsets, 0xFFFFFFFF = empty
	Raw         []byte
}

// ParseTypeSpec parses a TableTypeSpec chunk.
func ParseTypeSpec(buf []byte, off int) (*TableTypeSpec, error) {
	hdr, err := ReadHeader(buf, off)
	if err != nil {
		return nil, err
	}
	if hdr.Type != ChunkTableTypeSpec {
		return nil, fmt.Errorf("arsc: expected TypeSpec, got 0x%04X", hdr.Type)
	}
	body := buf[off : off+int(hdr.Size)]
	if len(body) < 16 {
		return nil, fmt.Errorf("arsc: typespec too small")
	}
	ts := &TableTypeSpec{
		Header:   hdr,
		TypeID:   body[8],
		EntryCount: binary.LittleEndian.Uint32(body[12:]),
	}
	return ts, nil
}

// ParseType parses a TableType chunk.
func ParseType(buf []byte, off int) (*TableType, error) {
	hdr, err := ReadHeader(buf, off)
	if err != nil {
		return nil, err
	}
	if hdr.Type != ChunkTableType {
		return nil, fmt.Errorf("arsc: expected Type, got 0x%04X", hdr.Type)
	}
	body := buf[off : off+int(hdr.Size)]
	if len(body) < 20 {
		return nil, fmt.Errorf("arsc: type too small")
	}

	t := &TableType{
		Header:      hdr,
		TypeID:      body[8],
		EntryCount:  binary.LittleEndian.Uint32(body[12:]),
		EntriesStart: binary.LittleEndian.Uint32(body[16:]),
	}

	// Config starts at offset 20.
	cfgSize := binary.LittleEndian.Uint32(body[20:])
	if 20+int(cfgSize) > len(body) {
		return nil, fmt.Errorf("arsc: config size overflow")
	}
	t.Config = ParseConfig(body[20 : 20+cfgSize])

	// Entry offsets start right after the config.
	offsetsStart := 20 + int(cfgSize)
	for i := uint32(0); i < t.EntryCount; i++ {
		if offsetsStart+int(i)*4+4 > len(body) {
			break
		}
		off := binary.LittleEndian.Uint32(body[offsetsStart+int(i)*4:])
		t.Offsets = append(t.Offsets, off)
	}
	t.Raw = body
	return t, nil
}
