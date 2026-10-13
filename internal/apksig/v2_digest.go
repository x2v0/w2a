package apksig

import (
	"crypto/sha256"
	"encoding/binary"
)

const v2ChunkSize = 1024 * 1024

// computeChunkedDigests hashes region [start,end) in 1 MB chunks.
// Chunk hash: SHA256(0xA5 || uint32LE(len) || chunk).
func computeChunkedDigests(apk []byte, start, end int) []byte {
	h := sha256.New()
	region := apk[start:end]
	pos := 0
	for pos < len(region) {
		cl := v2ChunkSize
		if len(region)-pos < cl {
			cl = len(region) - pos
		}
		ch := sha256.New()
		ch.Write([]byte{0xa5})
		var lb [4]byte
		binary.LittleEndian.PutUint32(lb[:], uint32(cl))
		ch.Write(lb[:])
		ch.Write(region[pos : pos+cl])
		h.Write(ch.Sum(nil))
		pos += cl
	}
	return h.Sum(nil)
}
