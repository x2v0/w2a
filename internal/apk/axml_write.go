package apk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// WriteAXML serialises an AXMLDocument back to binary AndroidManifest.xml
// format. It rebuilds the string pool from all strings used in the tree.
func WriteAXML(doc *AXMLDocument) ([]byte, error) {
	if doc == nil || doc.Root == nil {
		return nil, fmt.Errorf("axml: nil document")
	}

	pool := newPoolBuilder()
	collectStrings(doc.Root, pool)

	chunks := new(bytes.Buffer)
	if err := writeStartNamespace(chunks, pool, "android", "http://schemas.android.com/apk/res/android"); err != nil {
		return nil, err
	}
	if err := writeNode(chunks, doc.Root, pool); err != nil {
		return nil, err
	}
	if err := writeEndNamespace(chunks, pool, "android", "http://schemas.android.com/apk/res/android"); err != nil {
		return nil, err
	}

	poolChunk, err := buildStringPoolChunk(pool)
	if err != nil {
		return nil, err
	}

	fileBody := new(bytes.Buffer)
	fileBody.Write(poolChunk)
	fileBody.Write(chunks.Bytes())

	out := new(bytes.Buffer)
	writeU16(out, chunkXML)
	writeU16(out, 8)
	writeU32(out, uint32(8+fileBody.Len()))
	out.Write(fileBody.Bytes())
	return out.Bytes(), nil
}

type poolBuilder struct {
	index map[string]int
	list  []string
}

func newPoolBuilder() *poolBuilder {
	return &poolBuilder{index: map[string]int{}}
}

func (p *poolBuilder) add(s string) int {
	if idx, ok := p.index[s]; ok {
		return idx
	}
	idx := len(p.list)
	p.index[s] = idx
	p.list = append(p.list, s)
	return idx
}

func collectStrings(n *AXMLNode, pool *poolBuilder) {
	if n == nil {
		return
	}
	pool.add(n.Name)
	if n.Namespace != "" {
		pool.add(n.Namespace)
	}
	for _, a := range n.Attributes {
		pool.add(a.Name)
		if a.Namespace != "" {
			pool.add(a.Namespace)
		}
		if a.RawValue != "" {
			pool.add(a.RawValue)
		}
		if a.ValueStr != "" && a.ValueStr != a.RawValue {
			pool.add(a.ValueStr)
		}
	}
	for _, c := range n.Children {
		collectStrings(c, pool)
	}
}

func writeNode(buf *bytes.Buffer, n *AXMLNode, pool *poolBuilder) error {
	if err := writeStartElement(buf, n, pool); err != nil {
		return err
	}
	for _, c := range n.Children {
		if err := writeNode(buf, c, pool); err != nil {
			return err
		}
	}
	return writeEndElement(buf, n, pool)
}

func writeStartElement(buf *bytes.Buffer, n *AXMLNode, pool *poolBuilder) error {
	nameRef := pool.add(n.Name)
	nsRef := uint32(0xFFFFFFFF)
	if n.Namespace != "" {
		nsRef = uint32(pool.add(n.Namespace))
	}

	attrsBuf := new(bytes.Buffer)
	for _, a := range n.Attributes {
		writeAttribute(attrsBuf, a, pool)
	}

	body := new(bytes.Buffer)
	writeU16(body, chunkStartElement)
	writeU16(body, 16)
	writeU32(body, 0) // size placeholder
	writeU32(body, 0)
	writeU32(body, 0xFFFFFFFF)
	writeU32(body, nsRef)
	writeU32(body, uint32(nameRef))
	writeU16(body, 0x14)
	writeU16(body, 0x14)
	writeU16(body, uint16(len(n.Attributes)))
	writeU16(body, 0)
	writeU16(body, 0)
	writeU16(body, 0)
	body.Write(attrsBuf.Bytes())

	binary.LittleEndian.PutUint32(body.Bytes()[4:], uint32(body.Len()))
	buf.Write(body.Bytes())
	return nil
}

func writeAttribute(buf *bytes.Buffer, a AXMLAttribute, pool *poolBuilder) {
	nsRef := uint32(0xFFFFFFFF)
	if a.Namespace != "" {
		nsRef = uint32(pool.add(a.Namespace))
	}
	nameRef := uint32(pool.add(a.Name))
	rawRef := uint32(0xFFFFFFFF)
	if a.RawValue != "" {
		rawRef = uint32(pool.add(a.RawValue))
	}

	writeU32(buf, nsRef)
	writeU32(buf, nameRef)
	writeU32(buf, rawRef)

	writeU16(buf, 8)
	buf.WriteByte(0)

	vType := a.ValueType
	vData := a.ValueData
	if a.ValueStr != "" {
		vType = TypeString
		vData = uint32(pool.add(a.ValueStr))
	} else if vType == 0 {
		vType = TypeString
		if a.RawValue != "" {
			vData = uint32(pool.add(a.RawValue))
		}
	}
	buf.WriteByte(vType)
	writeU32(buf, vData)
}

func writeEndElement(buf *bytes.Buffer, n *AXMLNode, pool *poolBuilder) error {
	nameRef := uint32(pool.add(n.Name))
	nsRef := uint32(0xFFFFFFFF)
	if n.Namespace != "" {
		nsRef = uint32(pool.add(n.Namespace))
	}

	body := new(bytes.Buffer)
	writeU16(body, chunkEndElement)
	writeU16(body, 16)
	writeU32(body, 24)
	writeU32(body, 0)
	writeU32(body, 0xFFFFFFFF)
	writeU32(body, nsRef)
	writeU32(body, nameRef)
	buf.Write(body.Bytes())
	return nil
}

func writeStartNamespace(buf *bytes.Buffer, pool *poolBuilder, prefix, uri string) error {
	pRef := uint32(pool.add(prefix))
	uRef := uint32(pool.add(uri))
	b := new(bytes.Buffer)
	writeU16(b, chunkStartNamespace)
	writeU16(b, 16)
	writeU32(b, 24)
	writeU32(b, 0)
	writeU32(b, 0xFFFFFFFF)
	writeU32(b, pRef)
	writeU32(b, uRef)
	buf.Write(b.Bytes())
	return nil
}

func writeEndNamespace(buf *bytes.Buffer, pool *poolBuilder, prefix, uri string) error {
	pRef := uint32(pool.add(prefix))
	uRef := uint32(pool.add(uri))
	b := new(bytes.Buffer)
	writeU16(b, chunkEndNamespace)
	writeU16(b, 16)
	writeU32(b, 24)
	writeU32(b, 0)
	writeU32(b, 0xFFFFFFFF)
	writeU32(b, pRef)
	writeU32(b, uRef)
	buf.Write(b.Bytes())
	return nil
}

func buildStringPoolChunk(p *poolBuilder) ([]byte, error) {
	entries := make([][]byte, len(p.list))
	offset := 0
	offsets := make([]uint32, len(p.list))

	for i, s := range p.list {
		u16 := utf16.Encode([]rune(s))
		utf16Len := len(u16)
		bytelen := len(s)

		e := new(bytes.Buffer)
		// AXML UTF-8 string format: length fields are BIG-ENDIAN (u16 each).
		// AOSP string-pool entry: utf16len (varint) || bytelen (varint) ||
		// bytes || 0x00. Fixed 2-byte BE broke aapt2 ("manifest does not
		// start with <manifest> tag").
		writeVarLenBE(e, utf16Len)
		writeVarLenBE(e, bytelen)
		e.WriteString(s)
		e.WriteByte(0)
		entries[i] = e.Bytes()
		offsets[i] = uint32(offset)
		offset += len(entries[i])
	}

	offsetsBuf := new(bytes.Buffer)
	for _, o := range offsets {
		writeU32(offsetsBuf, o)
	}

	stringsBuf := new(bytes.Buffer)
	for _, e := range entries {
		stringsBuf.Write(e)
	}

	headerSize := 28
	stringsStart := headerSize + offsetsBuf.Len()
	totalSize := stringsStart + stringsBuf.Len()

	out := new(bytes.Buffer)
	writeU16(out, chunkStringPool)
	writeU16(out, uint16(headerSize))
	writeU32(out, uint32(totalSize))
	writeU32(out, uint32(len(p.list)))
	writeU32(out, 0)
	writeU32(out, 1<<8)
	writeU32(out, uint32(stringsStart))
	writeU32(out, 0)
	out.Write(offsetsBuf.Bytes())
	out.Write(stringsBuf.Bytes())
	return out.Bytes(), nil
}

func writeU16(buf *bytes.Buffer, v uint16) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}

func writeU32(buf *bytes.Buffer, v uint32) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}

// writeVarLenBE writes a big-endian variable-length integer as AOSP
// expects in string pool entries.
//
//	< 0x80        : single byte
//	>= 0x80       : first byte high-bit set with top 7 bits, second byte low 8
func writeVarLenBE(buf *bytes.Buffer, n int) {
	if n < 0x80 {
		buf.WriteByte(byte(n))
		return
	}
	buf.WriteByte(byte((n >> 8) | 0x80))
	buf.WriteByte(byte(n & 0xFF))
}
