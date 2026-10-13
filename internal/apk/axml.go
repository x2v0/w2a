// Package apk implements Android binary XML (AXML) parsing and writing,
// plus ZIP manipulation for APK archives.
package apk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

const (
	chunkStringPool     = 0x0001
	chunkXML            = 0x0003
	chunkResourceMap    = 0x0180
	chunkStartNamespace = 0x0100
	chunkEndNamespace   = 0x0101
	chunkStartElement   = 0x0102
	chunkEndElement     = 0x0103
	chunkCDATA          = 0x0104
)

const (
	TypeNull          = 0x00
	TypeReference     = 0x01
	TypeAttribute     = 0x02
	TypeString        = 0x03
	TypeFloat         = 0x04
	TypeDimension     = 0x05
	TypeFraction      = 0x06
	TypeDynamicRef    = 0x07
	TypeIntDec        = 0x10
	TypeIntHex        = 0x11
	TypeIntBoolean    = 0x12
	TypeIntColorARGB8 = 0x1c
	TypeIntColorRGB8  = 0x1d
	TypeIntColorARGB4 = 0x1e
	TypeIntColorRGB4  = 0x1f
)

type StringPool struct {
	IsUTF8  bool
	Strings []string
}

type AXMLAttribute struct {
	Namespace string
	Name      string
	RawValue  string
	ValueType uint8
	ValueData uint32
	ValueStr  string
}

type AXMLNode struct {
	Name       string
	Namespace  string
	Attributes []AXMLAttribute
	Children   []*AXMLNode
	Parent     *AXMLNode
	Text       string
}

type AXMLDocument struct {
	Root *AXMLNode
}

var (
	ErrAXMLHeader    = errors.New("axml: invalid header")
	ErrAXMLChunk     = errors.New("axml: invalid chunk")
	ErrAXMLStringRef = errors.New("axml: invalid string reference")
)

func readU16(buf []byte, off int) (uint16, error) {
	if off+2 > len(buf) {
		return 0, ErrAXMLHeader
	}
	return binary.LittleEndian.Uint16(buf[off:]), nil
}

func readU32(buf []byte, off int) (uint32, error) {
	if off+4 > len(buf) {
		return 0, ErrAXMLHeader
	}
	return binary.LittleEndian.Uint32(buf[off:]), nil
}

func decodeStringPool(buf []byte, off int) (*StringPool, error) {
	size, err := readU32(buf, off+4)
	if err != nil {
		return nil, err
	}
	if off+int(size) > len(buf) {
		return nil, ErrAXMLChunk
	}

	stringCount, _ := readU32(buf, off+8)
	flags, _ := readU32(buf, off+16)
	stringsStart, _ := readU32(buf, off+20)

	isUTF8 := flags&(1<<8) != 0
	headerSize, _ := readU16(buf, off+2)
	offsetsBase := off + int(headerSize)
	stringsBase := off + int(stringsStart)

	pool := &StringPool{IsUTF8: isUTF8, Strings: make([]string, 0, stringCount)}

	for i := uint32(0); i < stringCount; i++ {
		strOff, err := readU32(buf, offsetsBase+int(i)*4)
		if err != nil {
			return nil, err
		}
		strStart := stringsBase + int(strOff)
		var s string
		if isUTF8 {
			s, err = decodeUTF8String(buf, strStart)
		} else {
			s, err = decodeUTF16String(buf, strStart)
		}
		if err != nil {
			return nil, err
		}
		pool.Strings = append(pool.Strings, s)
	}
	return pool, nil
}

func decodeUTF8String(buf []byte, off int) (string, error) {
	if off+2 > len(buf) {
		return "", ErrAXMLChunk
	}
	off += 2
	if off+2 > len(buf) {
		return "", ErrAXMLChunk
	}
	byteLen := int(buf[off])<<8 | int(buf[off+1])
	off += 2
	if off+byteLen > len(buf) {
		return "", ErrAXMLChunk
	}
	return string(buf[off : off+byteLen]), nil
}

func decodeUTF16String(buf []byte, off int) (string, error) {
	if off+2 > len(buf) {
		return "", ErrAXMLChunk
	}
	strLen := int(binary.LittleEndian.Uint16(buf[off:]))
	off += 2
	if off+strLen*2 > len(buf) {
		return "", ErrAXMLChunk
	}
	u16 := make([]uint16, strLen)
	for i := 0; i < strLen; i++ {
		u16[i] = binary.LittleEndian.Uint16(buf[off+i*2:])
	}
	return string(utf16.Decode(u16)), nil
}

func (p *StringPool) StringAt(idx uint32) (string, error) {
	if idx == 0xFFFFFFFF {
		return "", nil
	}
	if int(idx) >= len(p.Strings) {
		return "", fmt.Errorf("%w: index %d, len %d", ErrAXMLStringRef, idx, len(p.Strings))
	}
	return p.Strings[idx], nil
}
