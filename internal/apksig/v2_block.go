package apksig

import (
	"bytes"
	"encoding/binary"
)

var apkSigBlockMagic = []byte{
	0x41, 0x50, 0x4B, 0x20, 0x53, 0x69, 0x67, 0x20,
	0x42, 0x6C, 0x6F, 0x63, 0x6B, 0x20, 0x34, 0x32,
}

const v2ID uint32 = 0x7109871a

type signingBlockPair struct {
	ID    uint32
	Value []byte
}

func buildSigningBlock(pairs []signingBlockPair) []byte {
	var body bytes.Buffer
	for _, p := range pairs {
		pairSize := uint64(4) + uint64(len(p.Value))
		binary.Write(&body, binary.LittleEndian, pairSize)
		binary.Write(&body, binary.LittleEndian, p.ID)
		body.Write(p.Value)
	}
	total := uint64(body.Len()) + 8 + 16
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, total)
	buf.Write(body.Bytes())
	binary.Write(&buf, binary.LittleEndian, total)
	buf.Write(apkSigBlockMagic)
	return buf.Bytes()
}
