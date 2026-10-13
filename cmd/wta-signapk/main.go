// Command wta-signapk signs an APK using pure Go (no apksigner/JDK).
package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"

	"github.com/x2v0/w2a/go/internal/apksig"
)

func main() {
	src := flag.String("in", "", "input APK path")
	dst := flag.String("out", "", "output APK path")
	ksPath := flag.String("keystore", "", "PEM keystore (private key + cert)")
	ksPass := flag.String("ks-pass", "", "keystore password (unused for PEM)")
	alias := flag.String("alias", "", "key alias (PKCS#12 only)")
	v1 := flag.Bool("v1", true, "enable JAR signing (V1)")
	v2 := flag.Bool("v2", false, "enable APK Signature Scheme v2 (WIP - has known issues)")
	flag.Parse()

	if *src == "" || *dst == "" || *ksPath == "" {
		fmt.Fprintln(os.Stderr, "usage: wta-signapk -in app.apk -out signed.apk -keystore release.pem")
		os.Exit(1)
	}

	cert, key, err := loadKeystore(*ksPath, *ksPass, *alias)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wta-signapk: %v\n", err)
		os.Exit(1)
	}

	opts := apksig.Options{Cert: cert, Key: key, V1: *v1, V2: *v2}
	if err := apksig.SignFile(*src, *dst, opts); err != nil {
		fmt.Fprintf(os.Stderr, "wta-signapk: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("signed: %s\n", *dst)
}

func loadKeystore(path, password, alias string) (*x509.Certificate, *rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var cert *x509.Certificate
	var key *rsa.PrivateKey

	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		switch block.Type {
		case "CERTIFICATE":
			if c, err := x509.ParseCertificate(block.Bytes); err == nil {
				cert = c
			}
		case "RSA PRIVATE KEY":
			if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
				key = k
			}
		case "PRIVATE KEY":
			if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
				if rk, ok := k.(*rsa.PrivateKey); ok {
					key = rk
				}
			}
		}
	}
	if cert == nil {
		return nil, nil, fmt.Errorf("no certificate found in %s", path)
	}
	if key == nil {
		return nil, nil, fmt.Errorf("no RSA private key found in %s", path)
	}
	return cert, key, nil
}
