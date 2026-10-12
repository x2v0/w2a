// Package apksig implements APK signing (v1 JAR + v2 APK Signature Scheme).
package apksig

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"math/big"
)

// oidSignedData is the PKCS#7 SignedData content type.
var (
	oidData          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidSignedData    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	oidSHA256        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	oidRSAEncryption = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
)

// contentInfo is a PKCS#7 ContentInfo.
type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

// signedData is the PKCS#7 SignedData structure.
type signedData struct {
	Version          int
	DigestAlgorithms []algorithmIdentifier `asn1:"set"`
	ContentInfo      contentInfo
	Certificates     asn1.RawValue `asn1:"optional,tag:0"`
	SignerInfos      []signerInfo  `asn1:"set"`
}

type algorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type signerInfo struct {
	Version                   int
	IssuerAndSerialNumber     issuerAndSerial
	DigestAlgorithm           algorithmIdentifier
	AuthenticatedAttributes   asn1.RawValue `asn1:"optional,tag:0"`
	DigestEncryptionAlgorithm algorithmIdentifier
	EncryptedDigest           []byte
	UnauthenticatedAttributes asn1.RawValue `asn1:"optional,tag:1"`
}

type issuerAndSerial struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
}

// SignPKCS7 creates a PKCS#7 detached signature over `data` using the
// given RSA private key and X.509 certificate.
func SignPKCS7(data []byte, cert *x509.Certificate, key *rsa.PrivateKey) ([]byte, error) {
	// 1. Compute SHA-256 of data.
	h := crypto.SHA256.New()
	h.Write(data)
	digest := h.Sum(nil)

	// 2. Sign the digest with RSA-PKCS#1v1.5.
	signature, err := rsa.SignPKCS1v15(nil, key, crypto.SHA256, digest)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: sign: %w", err)
	}

	// 3. Build the SignerInfo.
	si := signerInfo{
		Version: 1,
		IssuerAndSerialNumber: issuerAndSerial{
			Issuer:       asn1.RawValue{FullBytes: cert.RawIssuer},
			SerialNumber: cert.SerialNumber,
		},
		DigestAlgorithm: algorithmIdentifier{
			Algorithm: oidSHA256,
			Parameters: asn1.RawValue{
				Tag: 5, // NULL
			},
		},
		DigestEncryptionAlgorithm: algorithmIdentifier{
			Algorithm: oidRSAEncryption,
			Parameters: asn1.RawValue{
				Tag: 5,
			},
		},
		EncryptedDigest: signature,
	}

	sd := signedData{
		Version: 1,
		DigestAlgorithms: []algorithmIdentifier{
			{Algorithm: oidSHA256, Parameters: asn1.RawValue{Tag: 5}},
		},
		ContentInfo: contentInfo{
			ContentType: oidData,
		},
		Certificates: asn1.RawValue{
			Class:      2, // context-specific
			Tag:        0,
			IsCompound: true,
			Bytes:      cert.Raw,
		},
		SignerInfos: []signerInfo{si},
	}

	sdBytes, err := asn1.Marshal(sd)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: marshal signedData: %w", err)
	}

	ci := contentInfo{
		ContentType: oidSignedData,
		Content: asn1.RawValue{
			Class:      2,
			Tag:        0,
			IsCompound: true,
			Bytes:      sdBytes,
		},
	}
	return asn1.Marshal(ci)
}
