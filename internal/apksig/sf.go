package apksig

import (
	"bytes"
	"crypto/sha256"
)

// SFName is the signature file name inside META-INF (usually CERT.SF).
const SFName = "META-INF/CERT.SF"
const MFName = "META-INF/MANIFEST.MF"
const RSAName = "META-INF/CERT.RSA"

// BuildSF generates the .SF file from a manifest.
// The .SF contains SHA-256 digests of the *manifest sections* (not the
// full manifest file), plus the "SHA-256-Digest-Manifest" of the whole
// manifest.
func BuildSF(m *ManifestFile) []byte {
	mfBytes := m.Encode()
	mfDigest := sha256.Sum256(mfBytes)

	var buf bytes.Buffer
	buf.WriteString("Signature-Version: 1.0\r\n")
	buf.WriteString("Created-By: wta-buildapk (Go)\r\n")
	buf.WriteString("SHA-256-Digest-Manifest: " + base64StdNoPad(mfDigest[:]) + "\r\n")
	buf.WriteString("\r\n")

	for _, e := range m.Entries {
		sec := m.SectionOf(e.Name)
		if len(sec) == 0 {
			continue
		}
		d := sha256.Sum256(sec)
		buf.WriteString("Name: " + e.Name + "\r\n")
		buf.WriteString("SHA-256-Digest: " + base64StdNoPad(d[:]) + "\r\n")
		buf.WriteString("\r\n")
	}
	return buf.Bytes()
}
