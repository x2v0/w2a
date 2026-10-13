package arsc

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"
)

// WritePackage serialises a Package into binary chunk format.
func WritePackage(p *Package) []byte {
	// Serialize inner chunks first.
	inner := new(bytes.Buffer)

	// 1. TypeStrings pool.
	if p.TypeStrings != nil {
		inner.Write(WriteStringPool(p.TypeStrings))
	} else {
		inner.Write(WriteStringPool(&StringPool{IsUTF8: true}))
	}
	typeStringsOff := 0

	// 2. KeyStrings pool.
	if p.KeyStrings != nil {
		inner.Write(WriteStringPool(p.KeyStrings))
	} else {
		inner.Write(WriteStringPool(&StringPool{IsUTF8: true}))
	}
	keyStringsOff := inner.Len() - poolSize(p.KeyStrings) - poolSize(p.TypeStrings) + poolSize(p.TypeStrings)

	// Recompute offsets properly:
	inner.Reset()
	typeBytes := WriteStringPool(orEmpty(p.TypeStrings))
	inner.Write(typeBytes)
	keyBytes := WriteStringPool(orEmpty(p.KeyStrings))
	inner.Write(keyBytes)
	keyStringsOff = len(typeBytes)

	// 3. TypeSpecs (sorted by TypeID).
	typeIDs := make([]uint8, 0, len(p.TypeSpecs))
	for id := range p.TypeSpecs {
		typeIDs = append(typeIDs, id)
	}
	sortU8(typeIDs)
	for _, id := range typeIDs {
		inner.Write(WriteTypeSpec(p.TypeSpecs[id]))
	}

	// 4. Types (sorted by TypeID).
	typeIDs = make([]uint8, 0, len(p.Types))
	for id := range p.Types {
		typeIDs = append(typeIDs, id)
	}
	sortU8(typeIDs)
	for _, id := range typeIDs {
		for _, tt := range p.Types[id] {
			inner.Write(WriteType(tt))
		}
	}

	// Build package header.
	buf := new(bytes.Buffer)
	headerSize := 288
	writeU16TW(buf, ChunkTablePackage)
	writeU16TW(buf, uint16(headerSize))
	writeU32TW(buf, uint32(headerSize+inner.Len()))
	writeU32TW(buf, p.ID)

	// Name: 128 UTF-16 code units (256 bytes), null-terminated.
	nameBytes := encodePackageName(p.Name)
	buf.Write(nameBytes)

	// typeStrings / lastPublicType / keyStrings / lastPublicKey
	writeU32TW(buf, uint32(headerSize+typeStringsOff))
	writeU32TW(buf, 0) // lastPublicType (unused)
	writeU32TW(buf, uint32(headerSize+keyStringsOff))
	writeU32TW(buf, 0) // lastPublicKey
	writeU32TW(buf, 0) // typeIdOffset (added in API 26, required if headerSize=288)

	buf.Write(inner.Bytes())
	return buf.Bytes()
}

func orEmpty(p *StringPool) *StringPool {
	if p == nil {
		return &StringPool{IsUTF8: true}
	}
	return p
}

func poolSize(p *StringPool) int {
	if p == nil {
		return 0
	}
	return len(WriteStringPool(p))
}

// encodePackageName writes a 256-byte UTF-16LE null-padded name field.
func encodePackageName(name string) []byte {
	out := make([]byte, 256)
	u16 := utf16.Encode([]rune(name))
	for i, c := range u16 {
		if i*2+1 >= len(out) {
			break
		}
		binary.LittleEndian.PutUint16(out[i*2:], c)
	}
	return out
}

func sortU8(a []uint8) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

// WriteTable serialises a full Table into binary ARSC.
func WriteTable(t *Table) []byte {
	buf := new(bytes.Buffer)

	// 1. Global string pool.
	if t.GlobalPool != nil {
		buf.Write(WriteStringPool(t.GlobalPool))
	} else {
		buf.Write(WriteStringPool(&StringPool{IsUTF8: true}))
	}

	// 2. Packages.
	for _, pkg := range t.Packages {
		buf.Write(WritePackage(pkg))
	}

	inner := buf.Bytes()

	// 3. Table header.
	out := new(bytes.Buffer)
	headerSize := 12
	writeU16TW(out, ChunkTable)
	writeU16TW(out, uint16(headerSize))
	writeU32TW(out, uint32(headerSize+len(inner)))
	writeU32TW(out, uint32(len(t.Packages)))
	out.Write(inner)
	return out.Bytes()
}
