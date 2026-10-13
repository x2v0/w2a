package apksig

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func injectV2Block(apk []byte, signedData []byte) ([]byte, error) {
	eocdOff, eocd, err := findEOCD(apk)
	if err != nil {
		return nil, err
	}
	cdOff := int(binary.LittleEndian.Uint32(eocd[16:]))
	if cdOff < 0 || cdOff > eocdOff {
		return nil, fmt.Errorf("v2: bad CD offset %d", cdOff)
	}
	block := buildSigningBlock([]signingBlockPair{{ID: v2ID, Value: signedData}})
	var out bytes.Buffer
	out.Write(apk[:cdOff])
	out.Write(block)
	out.Write(apk[cdOff:eocdOff])
	newEOCD := append([]byte(nil), eocd...)
	binary.LittleEndian.PutUint32(newEOCD[16:], uint32(cdOff+len(block)))
	out.Write(newEOCD)
	return out.Bytes(), nil
}

func findEOCD(apk []byte) (int, []byte, error) {
	start := len(apk) - 65557
	if start < 0 {
		start = 0
	}
	for i := len(apk) - 22; i >= start; i-- {
		if apk[i] == 'P' && apk[i+1] == 'K' && apk[i+2] == 0x05 && apk[i+3] == 0x06 {
			cl := int(binary.LittleEndian.Uint16(apk[i+20:]))
			if i+22+cl != len(apk) {
				continue
			}
			return i, apk[i : i+22+cl], nil
		}
	}
	return 0, nil, fmt.Errorf("v2: EOCD not found")
}
