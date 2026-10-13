package apk

import (
	"fmt"
)

// AXMLChunkHeader is the common prefix for every chunk.
type AXMLChunkHeader struct {
	Type uint16
	Size uint32
}

func readChunkHeader(buf []byte, off int) (AXMLChunkHeader, error) {
	if off+8 > len(buf) {
		return AXMLChunkHeader{}, ErrAXMLHeader
	}
	typ, _ := readU16(buf, off)
	size, _ := readU32(buf, off+4)
	return AXMLChunkHeader{Type: typ, Size: size}, nil
}

// ParseAXML parses a binary AndroidManifest.xml into a tree.
func ParseAXML(buf []byte) (*AXMLDocument, error) {
	if len(buf) < 8 {
		return nil, ErrAXMLHeader
	}
	hdr, err := readChunkHeader(buf, 0)
	if err != nil {
		return nil, err
	}
	if hdr.Type != chunkXML {
		return nil, fmt.Errorf("axml: expected chunkXML (0x0003), got 0x%04X", hdr.Type)
	}

	pool, err := decodeStringPool(buf, 8)
	if err != nil {
		return nil, fmt.Errorf("axml: string pool: %w", err)
	}

	off := 8
	poolHdr, _ := readChunkHeader(buf, off)
	off += int(poolHdr.Size)

	doc := &AXMLDocument{}
	var stack []*AXMLNode

	for off+8 <= len(buf) {
		ch, err := readChunkHeader(buf, off)
		if err != nil {
			return nil, err
		}
		if ch.Size < 8 || off+int(ch.Size) > len(buf) {
			break
		}
		body := buf[off : off+int(ch.Size)]

		switch ch.Type {
		case chunkStartNamespace, chunkEndNamespace:
			// skip
		case chunkStartElement:
			node, err := parseStartElement(body, pool)
			if err != nil {
				return nil, err
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				node.Parent = parent
				parent.Children = append(parent.Children, node)
			} else if doc.Root == nil {
				doc.Root = node
			}
			stack = append(stack, node)
		case chunkEndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case chunkCDATA:
			if len(stack) > 0 && ch.Size >= 8+4+4 {
				textRef, _ := readU32(body, 8+4)
				txt, _ := pool.StringAt(textRef)
				stack[len(stack)-1].Text += txt
			}
		}
		off += int(ch.Size)
	}
	return doc, nil
}

func parseStartElement(body []byte, pool *StringPool) (*AXMLNode, error) {
	if len(body) < 36 {
		return nil, ErrAXMLChunk
	}
	nsRef, _ := readU32(body, 16)
	nameRef, _ := readU32(body, 20)
	attrCount, _ := readU16(body, 28)

	ns, _ := pool.StringAt(nsRef)
	name, _ := pool.StringAt(nameRef)

	node := &AXMLNode{
		Name:      name,
		Namespace: ns,
	}

	attrBase := 36
	for i := 0; i < int(attrCount); i++ {
		off := attrBase + i*20
		if off+20 > len(body) {
			return nil, ErrAXMLChunk
		}
		aNSRef, _ := readU32(body, off)
		aNameRef, _ := readU32(body, off+4)
		aRawRef, _ := readU32(body, off+8)
		aType := body[off+15]
		aData, _ := readU32(body, off+16)

		aNS, _ := pool.StringAt(aNSRef)
		aName, _ := pool.StringAt(aNameRef)
		aRaw, _ := pool.StringAt(aRawRef)

		attr := AXMLAttribute{
			Namespace: aNS,
			Name:      aName,
			RawValue:  aRaw,
			ValueType: aType,
			ValueData: aData,
		}
		if aType == TypeString {
			if s, err := pool.StringAt(aData); err == nil {
				attr.ValueStr = s
			}
		}
		node.Attributes = append(node.Attributes, attr)
	}
	return node, nil
}
