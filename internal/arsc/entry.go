package arsc

import (
	"encoding/binary"
	"fmt"
)

// Entry is a decoded resource entry.
type Entry struct {
	// For simple entries:
	Value    *ResValue
	// For complex (bag) entries:
	Bag      *Bag
	// Location metadata:
	Offset   uint32
}

// Bag is a "complex" entry — a map of attribute → value.
type Bag struct {
	ParentID uint32 // parent bag reference, or 0
	Items    []BagItem
}

// BagItem is one key-value pair inside a Bag.
type BagItem struct {
	Key   uint32   // attribute resource ID (string reference in many cases)
	Value ResValue
}

// ParseEntry reads one entry at the given offsets slice position.
// raw = body of the TableType chunk; entryOffset = relative offset from
// body + EntriesStart; the table gives us the absolute position.
func ParseEntry(body []byte, entriesStart int, entryOffset uint32) (*Entry, error) {
	if entryOffset == 0xFFFFFFFF {
		return nil, nil // empty entry
	}
	absOff := entriesStart + int(entryOffset)
	if absOff+8 > len(body) {
		return nil, fmt.Errorf("arsc: entry offset overflow")
	}

	flags := binary.LittleEndian.Uint16(body[absOff+2:])
	keyIdx := binary.LittleEndian.Uint32(body[absOff+4:])
	_ = keyIdx

	if flags&EntryFlagComplex != 0 {
		// Bag (complex) entry.
		if absOff+16 > len(body) {
			return nil, fmt.Errorf("arsc: bag too small")
		}
		parent := binary.LittleEndian.Uint32(body[absOff+8:])
		count := binary.LittleEndian.Uint32(body[absOff+12:])

		bag := &Bag{ParentID: parent}
		for i := uint32(0); i < count; i++ {
			itemOff := absOff + 16 + int(i)*12
			if itemOff+12 > len(body) {
				break
			}
			key := binary.LittleEndian.Uint32(body[itemOff:])
			valSize := binary.LittleEndian.Uint16(body[itemOff+4:])
			valType := body[itemOff+7]
			valData := binary.LittleEndian.Uint32(body[itemOff+8:])
			bag.Items = append(bag.Items, BagItem{
				Key: key,
				Value: ResValue{
					Size:     valSize,
					DataType: valType,
					Data:     valData,
				},
			})
		}
		return &Entry{Offset: entryOffset, Bag: bag}, nil
	}

	// Simple entry.
	// Layout: entry.size(2) entry.flags(2) entry.key(4) THEN Res_value(8):
	//         value.size(2) value.res0(1) value.dataType(1) value.data(4)
	if absOff+16 > len(body) {
		return nil, fmt.Errorf("arsc: simple entry too small")
	}
	valSize := binary.LittleEndian.Uint16(body[absOff+8:])
	valType := body[absOff+11]
	valData := binary.LittleEndian.Uint32(body[absOff+12:])
	return &Entry{
		Offset: entryOffset,
		Value: &ResValue{
			Size:     valSize,
			DataType: valType,
			Data:     valData,
		},
	}, nil
}
