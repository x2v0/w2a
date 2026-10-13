package apksig

import (
	"archive/zip"
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
	"os"
)

// Options controls APK signing.
type Options struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
	V1   bool
	V2   bool
}

// SignFile produces a signed APK.
//
// V1 (JAR signing) is fully implemented and verified with apksigner.
// V2 (APK Signature Scheme v2) is a work-in-progress: it will return an
// error if requested, instead of producing a non-installable APK.
func SignFile(src, dst string, opts Options) error {
	if opts.Cert == nil || opts.Key == nil {
		return fmt.Errorf("apksig: cert and key required")
	}
	if !opts.V1 && !opts.V2 {
		opts.V1 = true
	}
	if opts.V2 {
		return fmt.Errorf("apksig: V2 signing not yet complete — use -v1 only")
	}

	zr, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("apksig: open: %w", err)
	}
	defer zr.Close()

	type rawEntry struct {
		Name string
		Data []byte
	}
	var entries []rawEntry
	mf := &ManifestFile{}

	for _, f := range zr.File {
		if isSigFile(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		entries = append(entries, rawEntry{Name: f.Name, Data: data})
		if opts.V1 {
			mf.AddEntry(f.Name, data)
		}
	}

	out := new(bytes.Buffer)
	zw := zip.NewWriter(out)
	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.Name, Method: zip.Deflate}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		if _, err := w.Write(e.Data); err != nil {
			return err
		}
	}

	if opts.V1 {
		mfBytes := mf.Encode()
		hdr := &zip.FileHeader{Name: MFName, Method: zip.Store}
		w, _ := zw.CreateHeader(hdr)
		w.Write(mfBytes)

		sfBytes := BuildSF(mf)
		hdr = &zip.FileHeader{Name: SFName, Method: zip.Store}
		w, _ = zw.CreateHeader(hdr)
		w.Write(sfBytes)

		sig, err := SignPKCS7(sfBytes, opts.Cert, opts.Key)
		if err != nil {
			return err
		}
		hdr = &zip.FileHeader{Name: RSAName, Method: zip.Store}
		w, _ = zw.CreateHeader(hdr)
		w.Write(sig)
	}

	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(dst, out.Bytes(), 0o644)
}

func isSigFile(name string) bool {
	switch name {
	case "META-INF/MANIFEST.MF", "META-INF/CERT.SF", "META-INF/CERT.RSA":
		return true
	}
	if len(name) > 9 && name[:9] == "META-INF/" {
		rest := name[9:]
		for _, ext := range []string{".SF", ".RSA", ".DSA", ".EC"} {
			if len(rest) > len(ext) && rest[len(rest)-len(ext):] == ext {
				return true
			}
		}
	}
	return false
}
