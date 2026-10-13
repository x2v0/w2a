package arsc

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// StringPool holds a decoded string pool chunk.
type StringPool struct {
	IsUTF8  bool
	Strings []string
}

func ParseStringPool(buf []byte, off int) (*StringPool, error) {
	hdr, err := ReadHeader(buf, off)
	if err != nil {
		return nil, err
	}
	if hdr.Type != ChunkStringPool {
		return nil, fmt.Errorf("arsc: expected StringPool, got 0x%04X", hdr.Type)
	}
	body := buf[off : off+int(hdr.Size)]
	if len(body) < 28 {
		return nil, fmt.Errorf("arsc: string pool too small")
	}

	count := binary.LittleEndian.Uint32(body[8:])
	flags := binary.LittleEndian.Uint32(body[16:])
	stringsStart := binary.LittleEndian.Uint32(body[20:])

	isUTF8 := flags&(1<<8) != 0
	offsetsBase := int(hdr.HeaderSize)
	stringsBase := int(stringsStart)

	pool := &StringPool{IsUTF8: isUTF8, Strings: make([]string, 0, count)}

	for i := uint32(0); i < count; i++ {
		pos := offsetsBase + int(i)*4
		if pos+4 > len(body) {
			return nil, fmt.Errorf("arsc: offset %d out of bounds", i)
		}
		strOff := binary.LittleEndian.Uint32(body[pos:])
		strStart := stringsBase + int(strOff)

		var s string
		var err error
		if isUTF8 {
			s, err = decodeUTF8(body, strStart)
		} else {
			s, err = decodeUTF16(body, strStart)
		}
		if err != nil {
			return nil, fmt.Errorf("arsc: string %d: %w", i, err)
		}
		pool.Strings = append(pool.Strings, s)
	}
	return pool, nil
}

func (p *StringPool) At(idx uint32) string {
	if int(idx) >= len(p.Strings) {
		return ""
	}
	return p.Strings[idx]
}

// decodeVarLen reads a 1-or-2-byte length. Returns (value, bytesRead).
func decodeVarLen(buf []byte, off int) (int, int, error) {
	if off >= len(buf) {
		return 0, 0, fmt.Errorf("varlen: offset overflow")
	}
	b := buf[off]
	if b&0x80 != 0 {
		if off+1 >= len(buf) {
			return 0, 0, fmt.Errorf("varlen: 2-byte overflow")
		}
		return int(b&0x7F)<<8 | int(buf[off+1]), 2, nil
	}
	return int(b), 1, nil
}

// decodeUTF8 reads AXML-style UTF-8 string:
//   [varint utf16len] [varint bytelen] [bytes] [0x00]
func decodeUTF8(buf []byte, off int) (string, error) {
	_, n1, err := decodeVarLen(buf, off)
	if err != nil {
		return "", err
	}
	off += n1

	byteLen, n2, err := decodeVarLen(buf, off)
	if err != nil {
		return "", err
	}
	off += n2

	if off+byteLen > len(buf) {
		return "", fmt.Errorf("utf8: data overflow (need %d, have %d)", byteLen, len(buf)-off)
	}
	return string(buf[off : off+byteLen]), nil
}

// decodeUTF16 reads UTF-16LE string: [u16 len] [u16 chars...] [0x0000]
func decodeUTF16(buf []byte, off int) (string, error) {
	if off+2 > len(buf) {
		return "", fmt.Errorf("utf16: offset overflow")
	}
	strLen := int(binary.LittleEndian.Uint16(buf[off:]))
	off += 2
	if off+strLen*2 > len(buf) {
		return "", fmt.Errorf("utf16: data overflow")
	}
	u16 := make([]uint16, strLen)
	for i := 0; i < strLen; i++ {
		u16[i] = binary.LittleEndian.Uint16(buf[off+i*2:])
	}
	return string(utf16.Decode(u16)), nil
}
