package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/config"
	"gopkg.in/yaml.v3"
)

// signYAML mirrors `sign:` block in wta.yaml and global ~/.config/w2a/sign.yaml
type signYAML struct {
	Keystore string `yaml:"keystore"`
	KsPass   string `yaml:"ksPass,omitempty"`
	KeyAlias string `yaml:"keyAlias,omitempty"`
	KeyPass  string `yaml:"keyPass,omitempty"`
	V1       bool   `yaml:"v1,omitempty"`
	V2       bool   `yaml:"v2,omitempty"`
	V3       bool   `yaml:"v3,omitempty"`
	V4       bool   `yaml:"v4,omitempty"`
	MinSDK   int    `yaml:"minSdk,omitempty"`
}

func cmdSign(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	out := fs.String("o", "", "output APK (default: <apk>-signed.apk)")
	ks := fs.String("ks", "", "keystore path")
	ksPass := fs.String("ks-pass", "", "keystore password (pass:xxx)")
	keyAlias := fs.String("key-alias", "", "key alias")
	keyPass := fs.String("key-pass", "", "key password (pass:xxx)")
	v1 := fs.Bool("v1", false, "v1 scheme")
	v2 := fs.Bool("v2", false, "v2 scheme")
	v3 := fs.Bool("v3", false, "v3 scheme")
	v4 := fs.Bool("v4", false, "v4 scheme (writes .idsig)")
	minSdk := fs.Int("min-sdk", 0, "minSdk override")
	dryRun := fs.Bool("dry-run", false, "print apksigner command only")
	wtaYAML := fs.String("c", "", "wta.yaml path (default: next to APK)")
	noVerify := fs.Bool("no-verify", false, "skip apksigner verify")
	generate := fs.Bool("generate", false, "generate keystore instead of signing")
	cn := fs.String("cn", "", "DN: Common Name")
	org := fs.String("org", "", "DN: Organization (O)")
	ou := fs.String("ou", "", "DN: Organizational Unit (OU)")
	loc := fs.String("loc", "", "DN: Locality (L)")
	state := fs.String("state", "", "DN: State (ST)")
	country := fs.String("country", "", "DN: Country (C)")
	validity := fs.Int("validity", 10000, "cert validity (days)")
	keysize := fs.Int("keysize", 2048, "RSA keysize (bits)")
	force := fs.Bool("force", false, "overwrite existing keystore")
	noConfig := fs.Bool("no-config", false, "don't update ~/.config/w2a/sign.yaml")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wta sign [flags] <apk>")
		fmt.Fprintln(os.Stderr, "\nPriority: CLI → env → wta.yaml sign: → ~/.config/w2a/sign.yaml")
		fs.PrintDefaults()
	}
	// Go's flag package stops at first non-flag arg.
	// Reorder: "-flag" args first, positional last, so
	//   wta sign <apk> --dry-run
	// parses --dry-run correctly.
	{
		// Flags that take a separate value arg (not "-flag=value")
		// Flags that take a separate value arg.
		// Names use single-dash canonical form; comparison normalizes
		// --key-alias and -key-alias to the same key.
		valueFlags := map[string]bool{
			"o": true, "ks": true, "ks-pass": true,
			"key-alias": true, "key-pass": true,
			"min-sdk": true, "c": true,
			"cn": true, "org": true, "ou": true, "loc": true,
			"state": true, "country": true,
			"validity": true, "keysize": true,
		}
		norm := func(a string) string {
			return strings.TrimLeft(a, "-")
		}
		var flagArgs, posArgs []string
		for i := 0; i < len(args); i++ {
			a := args[i]
			if strings.HasPrefix(a, "-") {
				flagArgs = append(flagArgs, a)
				// skip "=" form
				if strings.Contains(a, "=") {
					continue
				}
				if valueFlags[norm(a)] && i+1 < len(args) {
					i++
					flagArgs = append(flagArgs, args[i])
				}
			} else {
				posArgs = append(posArgs, a)
			}
		}
		args = append(flagArgs, posArgs...)
	}
	fs.Parse(args)

	if *generate {
		if fs.NArg() > 0 {
			fmt.Fprintln(os.Stderr, "wta sign --generate: no <apk> expected")
			os.Exit(1)
		}
		ksPath := expandHome(*ks)
		if ksPath == "" {
			ksPath = expandHome("~/.config/w2a/keystore.p12")
		}
		storePass := strings.TrimPrefix(*ksPass, "pass:")
		if storePass == "" {
			storePass = os.Getenv("WTA_KS_PASS")
		}
		aliasV := *keyAlias
		if aliasV == "" {
			aliasV = "wta"
		}
		keyPassV := strings.TrimPrefix(*keyPass, "pass:")
		if keyPassV == "" {
			keyPassV = os.Getenv("WTA_KEY_PASS")
		}
		err := generateKeystore(generateOpts{
			Out: ksPath, StorePass: storePass,
			KeyAlias: aliasV, KeyPass: keyPassV,
			CN: *cn, O: *org, OU: *ou,
			L: *loc, ST: *state, C: *country,
			Validity: *validity, KeySize: *keysize,
			Force: *force, NoConfig: *noConfig,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "wta sign --generate: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}
	apkPath := fs.Arg(0)
	if _, err := os.Stat(apkPath); err != nil {
		fmt.Fprintf(os.Stderr, "wta sign: %v\n", err)
		os.Exit(1)
	}

	global := loadGlobalSignConfig()
	project := loadProjectSignConfig(*wtaYAML, apkPath)

	pick := func(cli, env, proj, glob, def string) string {
		if cli != "" {
			return cli
		}
		if env != "" {
			return env
		}
		if proj != "" {
			return proj
		}
		if glob != "" {
			return glob
		}
		return def
	}

	keystore := expandHome(pick(*ks, os.Getenv("WTA_KEYSTORE"), project.Keystore, global.Keystore, "~/.config/w2a/keystore.p12"))
	ksPassV := normalizePass(pick(*ksPass, os.Getenv("WTA_KS_PASS"), project.KsPass, global.KsPass, ""))
	aliasV := pick(*keyAlias, os.Getenv("WTA_KEY_ALIAS"), project.KeyAlias, global.KeyAlias, "")
	keyPassV := normalizePass(pick(*keyPass, os.Getenv("WTA_KEY_PASS"), project.KeyPass, global.KeyPass, ""))

	useV1 := *v1 || project.V1 || global.V1
	useV2 := *v2 || project.V2 || global.V2
	useV3 := *v3 || project.V3 || global.V3
	useV4 := *v4 || project.V4 || global.V4

	// Only CLI flags trigger explicit mode. Config schemes are fallbacks
	// (they fill in the truthy set once triggered), but they cannot force
	// --vX-signing-enabled false, which breaks verify on minSdk<24 APKs.
	explicitScheme := *v1 || *v2 || *v3 || *v4

	minSdkV := *minSdk
	if minSdkV == 0 {
		minSdkV = project.MinSDK
	}
	if minSdkV == 0 {
		minSdkV = global.MinSDK
	}

	outputPath := *out
	if outputPath == "" {
		outputPath = strings.TrimSuffix(apkPath, ".apk") + "-signed.apk"
	}

	if _, err := os.Stat(keystore); err != nil {
		fmt.Fprintf(os.Stderr, "wta sign: keystore not found: %s\n", keystore)
		fmt.Fprintln(os.Stderr, "  --ks /path | env WTA_KEYSTORE | wta.yaml sign.keystore | ~/.config/w2a/sign.yaml")
		os.Exit(1)
	}
	if ksPassV == "" || aliasV == "" || keyPassV == "" {
		fmt.Fprintln(os.Stderr, "wta sign: ksPass / keyAlias / keyPass required")
		fmt.Fprintln(os.Stderr, "  flags, env, wta.yaml sign: or ~/.config/w2a/sign.yaml")
		os.Exit(1)
	}

	cmdArgs := []string{
		"sign",
		"--ks", keystore,
		"--ks-pass", ksPassV,
		"--ks-key-alias", aliasV,
		"--key-pass", keyPassV,
		"--out", outputPath,
	}
	if explicitScheme {
		// Emit only enabled schemes; leave others to apksigner auto.
		// This avoids forcing --vX-signing-enabled false, which would
		// break v1 for minSdk<24 APKs (Missing META-INF/MANIFEST.MF).
		if useV1 {
			cmdArgs = append(cmdArgs, "--v1-signing-enabled", "true")
		}
		if useV2 {
			cmdArgs = append(cmdArgs, "--v2-signing-enabled", "true")
		}
		if useV3 {
			cmdArgs = append(cmdArgs, "--v3-signing-enabled", "true")
		}
		if useV4 {
			cmdArgs = append(cmdArgs, "--v4-signing-enabled", "true")
		}
	}
	if minSdkV > 0 {
		cmdArgs = append(cmdArgs, "--min-sdk-version", fmt.Sprintf("%d", minSdkV))
	}
	cmdArgs = append(cmdArgs, apkPath)

	if *dryRun {
		fmt.Println("apksigner", redactArgs(cmdArgs))
		return
	}

	schemesMsg := "auto (per APK minSdk)"
	if explicitScheme {
		schemesMsg = fmt.Sprintf("v1=%t v2=%t v3=%t v4=%t", useV1, useV2, useV3, useV4)
	}
	fmt.Fprintf(os.Stderr, "→ signing: %s\n  keystore: %s\n  alias: %s\n  schemes: %s\n  out: %s\n",
		apkPath, keystore, aliasV, schemesMsg, outputPath)

	cmd := exec.Command("apksigner", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "wta sign: apksigner failed: %v\n", err)
		os.Exit(1)
	}

	if !*noVerify {
		if err := verifyAPK(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "wta sign: verify failed: %v\n", err)
			os.Exit(1)
		}
	}

	st, _ := os.Stat(outputPath)
	fmt.Printf("[OK] signed: %s (%d bytes)\n", outputPath, st.Size())
}

func loadGlobalSignConfig() signYAML {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".config", "w2a", "sign.yaml"))
	if err != nil {
		return signYAML{}
	}
	var c signYAML
	_ = yaml.Unmarshal(data, &c)
	return c
}

func loadProjectSignConfig(yamlPath, apkPath string) signYAML {
	candidate := yamlPath
	if candidate == "" {
		candidate = filepath.Join(filepath.Dir(apkPath), "wta.yaml")
	}
	if _, err := os.Stat(candidate); err != nil {
		return signYAML{}
	}
	cfg, err := config.Load(candidate)
	if err != nil {
		return signYAML{}
	}
	s := cfg.Sign
	return signYAML{
		Keystore: s.Keystore, KsPass: s.KsPass,
		KeyAlias: s.KeyAlias, KeyPass: s.KeyPass,
		V1: s.V1, V2: s.V2, V3: s.V3, V4: s.V4,
		MinSDK: s.MinSDK,
	}
}

func verifyAPK(path string) error {
	out, err := exec.Command("apksigner", "verify", path).CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "WARNING:") || strings.TrimSpace(line) == "" {
			continue
		}
		fmt.Println(line)
	}
	return err
}

func normalizePass(p string) string {
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "pass:") || strings.HasPrefix(p, "env:") ||
		strings.HasPrefix(p, "file:") || strings.HasPrefix(p, "stdin") {
		return p
	}
	return "pass:" + p
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func redactArgs(args []string) string {
	var b strings.Builder
	for i, a := range args {
		if i > 0 && (args[i-1] == "--ks-pass" || args[i-1] == "--key-pass") {
			b.WriteString("pass:***")
		} else {
			b.WriteString(a)
		}
		if i < len(args)-1 {
			b.WriteByte(' ')
		}
	}
	return b.String()
}
