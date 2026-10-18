package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/signing"
	"gopkg.in/yaml.v3"
)

type generateOpts struct {
	Out       string
	StorePass string
	KeyAlias  string
	KeyPass   string
	CN, O, OU string
	L, ST, C  string
	Validity  int
	KeySize   int
	Force     bool
	NoConfig  bool
}

func generateKeystore(o generateOpts) error {
	if _, err := os.Stat(o.Out); err == nil && !o.Force {
		return fmt.Errorf("keystore already exists: %s (use --force to overwrite)", o.Out)
	}
	if dir := filepath.Dir(o.Out); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o700)
	}

	genPass := false
	if o.StorePass == "" {
		p, err := randomPassword(24)
		if err != nil {
			return err
		}
		o.StorePass = p
		genPass = true
	}
	if o.KeyPass == "" {
		o.KeyPass = o.StorePass
	}
	if o.KeyAlias == "" {
		o.KeyAlias = "wta"
	}
	if o.Validity == 0 {
		o.Validity = 10000
	}
	if o.KeySize == 0 {
		o.KeySize = 2048
	}

	var dn []string
	add := func(k, v string) {
		if v != "" {
			dn = append(dn, k+"="+v)
		}
	}
	add("CN", o.CN)
	add("O", o.O)
	add("OU", o.OU)
	add("L", o.L)
	add("ST", o.ST)
	add("C", o.C)
	if len(dn) == 0 {
		dn = append(dn, "CN=wta")
	}
	dname := strings.Join(dn, ", ")

	args := []string{
		"-genkeypair",
		"-keystore", o.Out,
		"-storetype", "PKCS12",
		"-storepass", o.StorePass,
		"-keypass", o.KeyPass,
		"-alias", o.KeyAlias,
		"-keyalg", "RSA",
		"-keysize", fmt.Sprintf("%d", o.KeySize),
		"-validity", fmt.Sprintf("%d", o.Validity),
		"-dname", dname,
	}

	fmt.Fprintf(os.Stderr, "→ generating: %s\n  alias:    %s\n  DN:       %s\n  validity: %d days\n  keysize:  %d\n",
		o.Out, o.KeyAlias, dname, o.Validity, o.KeySize)

	cmd := exec.Command("keytool", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keytool failed: %w", err)
	}

	_ = os.Chmod(o.Out, 0o600)

	fp := fingerprint(o.Out, o.StorePass)

	if !o.NoConfig {
		if err := updateSignYAML(o.Out, o.StorePass, o.KeyAlias, o.KeyPass); err != nil {
			fmt.Fprintf(os.Stderr, "warn: failed to update sign.yaml: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "\n[OK] keystore: %s\n", o.Out)
	if fp != "" {
		fmt.Fprintf(os.Stderr, "     SHA256: %s\n", fp)
	}
	if genPass {
		fmt.Fprintf(os.Stderr, "\n     store pass: %s\n     key pass:   %s\n", o.StorePass, o.KeyPass)
	}
	fmt.Fprintln(os.Stderr, "\n⚠  BACKUP THIS FILE + PASSWORD!")
	fmt.Fprintln(os.Stderr, "   Lost keystore = lost ability to update APK on users' devices.")
	return nil
}

func randomPassword(n int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var b strings.Builder
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b.WriteByte(alphabet[idx.Int64()])
	}
	return b.String(), nil
}

func fingerprint(keystore, pass string) string {
	out, err := exec.Command("keytool", "-list", "-v",
		"-keystore", keystore,
		"-storetype", "PKCS12",
		"-storepass", pass).CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		for _, key := range []string{"SHA-256:", "SHA256:", "SHA-256 :", "SHA256 :"} {
			if strings.HasPrefix(line, key) {
				return strings.TrimSpace(strings.TrimPrefix(line, key))
			}
		}
	}
	return ""
}

func updateSignYAML(keystore, ksPass, alias, keyPass string) error {
	path := signing.GlobalConfigPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)

	c := signing.Config{}
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, &c)
	}
	c.Keystore = keystore
	c.KsPass = "pass:" + ksPass
	c.KeyAlias = alias
	c.KeyPass = "pass:" + keyPass

	data, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "→ updated: %s\n", path)
	return nil
}
