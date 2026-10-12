package apksig

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"fmt"
)

// Signature algorithm ID: RSASSA-PKCS1-v1_5 with SHA2-256.
const sigAlgRSAPKCS1SHA256 = 0x0103

// lengthPrefixed returns uint32LE(len) || data.
func lengthPrefixed(data []byte) []byte {
	out := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(data)))
	copy(out[4:], data)
	return out
}

// buildSignedData builds the "signed data" contents:
//
//	LP(digests) || LP(certificates) || LP(additional_attributes)
func buildSignedData(apkDigest []byte, certDER []byte) []byte {
	// digest record: uint32LE(sigAlg) || LP(digest)
	digestRecord := make([]byte, 4+4+len(apkDigest))
	digestRecord[0] = 0x03
	digestRecord[1] = 0x01
	digestRecord[2] = 0x00
	digestRecord[3] = 0x00
	// LP(digest) = uint32LE(len) || digest
	digestRecord[4] = byte(len(apkDigest))
	digestRecord[5] = byte(len(apkDigest) >> 8)
	digestRecord[6] = byte(len(apkDigest) >> 16)
	digestRecord[7] = byte(len(apkDigest) >> 24)
	copy(digestRecord[8:], apkDigest)

	// digests: LP(LP(digestRecord)) — outer LP for sequence, inner LP for the item.
	digestsContent := lengthPrefixed(lengthPrefixed(digestRecord))

	// certificates: LP(LP(certDER)) — outer LP for the list, inner LP for each cert.
	certsContent := lengthPrefixed(lengthPrefixed(certDER))

	// additional attributes: empty list -> LP(empty) = 4 zero bytes.
	attrsContent := lengthPrefixed([]byte{})

	// Signed data layout:
	//   LP(digests) || LP(certs) || LP(attrs)
	// where digestsContent is ALREADY LP(record) — no extra wrap.
	var sd []byte
	sd = append(sd, digestsContent...)
	sd = append(sd, certsContent...)
	sd = append(sd, attrsContent...)
	return sd
}

// buildSigner assembles a full signer structure:
//
//	LP(signed_data) || LP(signatures) || LP(public_key)
func buildSigner(apkDigest []byte, cert *x509.Certificate, key *rsa.PrivateKey) ([]byte, error) {
	sd := buildSignedData(apkDigest, cert.Raw)

	// Compute signature over: 0x5a || uint32LE(len(sd)) || sd
	h := sha256.New()
	h.Write([]byte{0x5a})
	var lb [4]byte
	binary.LittleEndian.PutUint32(lb[:], uint32(len(sd)))
	h.Write(lb[:])
	h.Write(sd)
	signature, err := rsa.SignPKCS1v15(nil, key, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return nil, fmt.Errorf("v2: sign: %w", err)
	}

	// signature record: uint32LE(sigAlg) || LP(signature)
	sigRecord := make([]byte, 4+4+len(signature))
	sigRecord[0] = 0x03
	sigRecord[1] = 0x01
	sigRecord[2] = 0x00
	sigRecord[3] = 0x00
	sigRecord[4] = byte(len(signature))
	sigRecord[5] = byte(len(signature) >> 8)
	sigRecord[6] = byte(len(signature) >> 16)
	sigRecord[7] = byte(len(signature) >> 24)
	copy(sigRecord[8:], signature)

	// signatures list content: LP(sigRecord)
	signaturesContent := lengthPrefixed(sigRecord)

	// public key: raw SubjectPublicKeyInfo DER
	publicKey := cert.RawSubjectPublicKeyInfo

	var signer []byte
	signer = append(signer, lengthPrefixed(sd)...)
	signer = append(signer, lengthPrefixed(signaturesContent)...)
	signer = append(signer, lengthPrefixed(publicKey)...)
	return signer, nil
}

// buildV2SignedData returns the v2 pair value:
//
//	LP(signer)   (a sequence containing one signer)
func buildV2SignedData(apkDigest []byte, cert *x509.Certificate, key *rsa.PrivateKey) ([]byte, error) {
	signer, err := buildSigner(apkDigest, cert, key)
	if err != nil {
		return nil, err
	}
	// Two levels of LP: one for the "signers sequence" wrapper, one for
	// the individual signer record.
	return lengthPrefixed(lengthPrefixed(signer)), nil
}
