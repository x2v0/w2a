package arsc

import (
	"bytes"
	"encoding/binary"
)

// WriteTypeSpec serialises a TableTypeSpec chunk.
func WriteTypeSpec(ts *TableTypeSpec) []byte {
	buf := new(bytes.Buffer)
	writeU16TW(buf, ChunkTableTypeSpec)
	writeU16TW(buf, 16) // header size
	total := 16 + int(ts.EntryCount)*4
	writeU32TW(buf, uint32(total))
	buf.WriteByte(ts.TypeID)
	buf.WriteByte(0)          // res0
	writeU16TW(buf, 0)        // res1
	writeU32TW(buf, ts.EntryCount)

	// Entries array (u32 per entry). We reuse the ones from the original
	// if present, otherwise zero-fill.
	for i := uint32(0); i < ts.EntryCount; i++ {
		writeU32TW(buf, 0) // config flags for entry i (unused by most parsers)
	}
	return buf.Bytes()
}

// WriteType serialises a TableType chunk.
func WriteType(t *TableType) []byte {
	buf := new(bytes.Buffer)

	// First compute the size of the config, offsets and entries.
	configBytes := t.Config.Raw
	if len(configBytes) == 0 {
		// Fall back to a minimal 64-byte config.
		configBytes = make([]byte, 64)
		binary.LittleEndian.PutUint32(configBytes, 64)
	}

	headerSize := 20
	offsetsSize := int(t.EntryCount) * 4
	entriesStart := headerSize + len(configBytes) + offsetsSize

	// Copy the entire entries block verbatim. Individual entry copying is
	// broken for complex (bag) entries because their size isn't equal to the
	// next entry's offset delta.
	entriesBuf := new(bytes.Buffer)
	offsets := make([]uint32, t.EntryCount)
	for i := uint32(0); i < t.EntryCount; i++ {
		if int(i) < len(t.Offsets) {
			offsets[i] = t.Offsets[i]
		} else {
			offsets[i] = 0xFFFFFFFF
		}
	}
	if int(t.EntriesStart) <= len(t.Raw) {
		entriesBuf.Write(t.Raw[t.EntriesStart:])
	}

	totalSize := entriesStart + entriesBuf.Len()

	// Write header.
	writeU16TW(buf, ChunkTableType)
	writeU16TW(buf, uint16(headerSize))
	writeU32TW(buf, uint32(totalSize))
	buf.WriteByte(t.TypeID)
	buf.WriteByte(0)
	writeU16TW(buf, 0)
	writeU32TW(buf, t.EntryCount)
	writeU32TW(buf, uint32(entriesStart))

	// Config.
	buf.Write(configBytes)

	// Offsets.
	for _, off := range offsets {
		writeU32TW(buf, off)
	}

	// Entries.
	buf.Write(entriesBuf.Bytes())
	return buf.Bytes()
}

func writeU16TW(buf *bytes.Buffer, v uint16) { _ = binary.Write(buf, binary.LittleEndian, v) }
func writeU32TW(buf *bytes.Buffer, v uint32) { _ = binary.Write(buf, binary.LittleEndian, v) }
