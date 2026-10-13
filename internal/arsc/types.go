// Package arsc implements Android binary resource table (resources.arsc)
// parsing. It is read-only for now: new resources are compiled via aapt2
// as a subprocess, and only merge/query operations happen in Go.
package arsc

import (
	"encoding/binary"
	"errors"
)

// Chunk types found in resources.arsc.
const (
	ChunkNull          = 0x0000
	ChunkStringPool    = 0x0001
	ChunkTable         = 0x0002
	ChunkXML           = 0x0003
	ChunkTablePackage  = 0x0200
	ChunkTableType     = 0x0201
	ChunkTableTypeSpec = 0x0202
	ChunkTableLibrary  = 0x0203
	ChunkTableOverlay  = 0x0204
)

// Res_value data types (duplicated from apk package to keep arsc standalone).
const (
	TypeNull          = 0x00
	TypeReference     = 0x01
	TypeAttribute     = 0x02
	TypeString        = 0x03
	TypeFloat         = 0x04
	TypeDimension     = 0x05
	TypeFraction      = 0x06
	TypeDynamicRef    = 0x07
	TypeDynamicAttr   = 0x08
	TypeIntDec        = 0x10
	TypeIntHex        = 0x11
	TypeIntBoolean    = 0x12
	TypeIntColorARGB8 = 0x1c
	TypeIntColorRGB8  = 0x1d
	TypeIntColorARGB4 = 0x1e
	TypeIntColorRGB4  = 0x1f
)

// ChunkHeader is the common prefix of every ARSC chunk.
type ChunkHeader struct {
	Type       uint16
	HeaderSize uint16
	Size       uint32
}

// ReadHeader reads a chunk header at the given offset.
func ReadHeader(buf []byte, off int) (ChunkHeader, error) {
	if off+8 > len(buf) {
		return ChunkHeader{}, errors.New("arsc: header overflow")
	}
	return ChunkHeader{
		Type:       binary.LittleEndian.Uint16(buf[off:]),
		HeaderSize: binary.LittleEndian.Uint16(buf[off+2:]),
		Size:       binary.LittleEndian.Uint32(buf[off+4:]),
	}, nil
}

// Res_value is a typed resource value (usually 8 bytes).
type ResValue struct {
	Size     uint16
	Res0     uint8
	DataType uint8
	Data     uint32
}

// ResTableRef is a reference to another resource: package<<24 | type<<16 | entry.
type ResTableRef uint32

func (r ResTableRef) IsValid() bool  { return uint32(r) != 0 }
func (r ResTableRef) Package() uint8 { return uint8(uint32(r) >> 24) }
func (r ResTableRef) Type() uint8    { return uint8((uint32(r) >> 16) & 0xFF) }
func (r ResTableRef) Entry() uint16  { return uint16(uint32(r) & 0xFFFF) }

// PackageID constants.
const (
	PackageIDApp   = 0x7F // normal user app
	PackageIDFrame = 0x01 // framework (android)
)

// Entry flags.
const (
	EntryFlagComplex     = 0x0001 // entry is a bag (map)
	EntryFlagPublic      = 0x0002
	EntryFlagWeak        = 0x0004
	EntryFlagCompact     = 0x0008 // no Res_value header, just 4-byte data
	EntryFlagCompactOffsetMask = 0xFFFFFFFC
)
