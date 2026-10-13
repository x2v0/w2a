package arsc

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"
)

// WriteStringPool serialises a StringPool into binary chunk format.
// Encodes strings in UTF-8 or UTF-16 depending on pool.IsUTF8.
func WriteStringPool(p *StringPool) []byte {
	// 1. Encode each string into bytes.
	encoded := make([][]byte, len(p.Strings))
	offsets := make([]uint32, len(p.Strings))
	total := 0

	for i, s := range p.Strings {
		var e []byte
		if p.IsUTF8 {
			e = encodeUTF8Entry(s)
		} else {
			e = encodeUTF16Entry(s)
		}
		encoded[i] = e
		offsets[i] = uint32(total)
		total += len(e)
	}

	// 2. Compute layout.
	headerSize := 28
	offsetBytes := len(offsets) * 4
	stringsStart := headerSize + offsetBytes
	pad := (4 - (stringsStart % 4)) % 4 // strings start on 4-byte boundary
	stringsStart += pad
	totalSize := stringsStart + total

	// 3. Write.
	buf := new(bytes.Buffer)
	writeU16SP(buf, ChunkStringPool)
	writeU16SP(buf, uint16(headerSize))
	writeU32SP(buf, uint32(totalSize))
	writeU32SP(buf, uint32(len(p.Strings))) // string count
	writeU32SP(buf, 0)                      // style count
	flags := uint32(0)
	if p.IsUTF8 {
		flags = 1 << 8
	}
	writeU32SP(buf, flags)
	writeU32SP(buf, uint32(stringsStart))
	writeU32SP(buf, 0) // styles start

	// Offsets array.
	for _, off := range offsets {
		writeU32SP(buf, off)
	}
	// Padding.
	for i := 0; i < pad; i++ {
		buf.WriteByte(0)
	}
	// Strings data.
	for _, e := range encoded {
		buf.Write(e)
	}
	return buf.Bytes()
}

// encodeUTF8Entry encodes "hello" as:
//   [varint utf16len] [varint bytelen] [bytes] [0x00]
func encodeUTF8Entry(s string) []byte {
	u16 := utf16.Encode([]rune(s))
	utf16Len := len(u16)
	byteLen := len(s)

	buf := new(bytes.Buffer)
	writeVarLenSP(buf, utf16Len)
	writeVarLenSP(buf, byteLen)
	buf.WriteString(s)
	buf.WriteByte(0)
	return buf.Bytes()
}

// encodeUTF16Entry encodes "hello" as: [u16 len] [u16 chars...] [0x0000]
func encodeUTF16Entry(s string) []byte {
	u16 := utf16.Encode([]rune(s))
	buf := new(bytes.Buffer)
	writeU16SP(buf, uint16(len(u16)))
	for _, c := range u16 {
		writeU16SP(buf, c)
	}
	writeU16SP(buf, 0)
	return buf.Bytes()
}

// writeVarLenSP writes 1 or 2 bytes (big-endian length).
func writeVarLenSP(buf *bytes.Buffer, n int) {
	if n < 0x80 {
		buf.WriteByte(byte(n))
		return
	}
	// 2-byte form: high bit set on first byte.
	buf.WriteByte(byte((n >> 8) | 0x80))
	buf.WriteByte(byte(n & 0xFF))
}

// writeU16SP writes a little-endian uint16.
func writeU16SP(buf *bytes.Buffer, v uint16) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}

// writeU32SP writes a little-endian uint32.
func writeU32SP(buf *bytes.Buffer, v uint32) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}
