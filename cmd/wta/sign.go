package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/signing"
)

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
	cn := fs.String("cn", "", "DN: Common Name (for --generate)")
	org := fs.String("org", "", "DN: Organization (for --generate)")
	ou := fs.String("ou", "", "DN: Organizational Unit (for --generate)")
	loc := fs.String("loc", "", "DN: Locality (for --generate)")
	state := fs.String("state", "", "DN: State (for --generate)")
	country := fs.String("country", "", "DN: Country (for --generate)")
	validity := fs.Int("validity", 10000, "cert validity days (for --generate)")
	keysize := fs.Int("keysize", 2048, "RSA keysize (for --generate)")
	force := fs.Bool("force", false, "overwrite existing keystore (for --generate)")
	noConfig := fs.Bool("no-config", false, "don't update sign.yaml (for --generate)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wta sign [flags] <apk>")
		fmt.Fprintln(os.Stderr, "\nPriority: CLI → env → wta.yaml sign: → ~/.config/w2a/sign.yaml")
		fs.PrintDefaults()
	}

	// Go's flag package stops at first non-flag arg.
	// Reorder: "-flag" args first, positional last.
	{
		valueFlags := map[string]bool{
			"o": true, "ks": true, "ks-pass": true,
			"key-alias": true, "key-pass": true,
			"min-sdk": true, "c": true,
			"cn": true, "org": true, "ou": true, "loc": true,
			"state": true, "country": true,
			"validity": true, "keysize": true,
		}
		norm := func(a string) string { return strings.TrimLeft(a, "-") }
		var flagArgs, posArgs []string
		for i := 0; i < len(args); i++ {
			a := args[i]
			if strings.HasPrefix(a, "-") {
				flagArgs = append(flagArgs, a)
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
		ksPath := signing.ExpandHome(*ks)
		if ksPath == "" {
			ksPath = signing.ExpandHome("~/.config/w2a/keystore.p12")
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

	// Resolve config sources
	global := signing.LoadGlobal()
	projectYAML := *wtaYAML
	if projectYAML == "" {
		projectYAML = filepath.Join(filepath.Dir(apkPath), "wta.yaml")
	}
	project := signing.LoadProject(projectYAML)

	cliOpts := signing.Options{
		Keystore: *ks, KsPass: *ksPass,
		KeyAlias: *keyAlias, KeyPass: *keyPass,
		V1: *v1, V2: *v2, V3: *v3, V4: *v4,
		MinSDK:         *minSdk,
		ExplicitScheme: *v1 || *v2 || *v3 || *v4,
	}
	opts := signing.Resolve(cliOpts, project, global)

	outputPath := *out
	if outputPath == "" {
		outputPath = strings.TrimSuffix(apkPath, ".apk") + "-signed.apk"
	}

	if *dryRun {
		// build apksigner args for display only
		args := []string{
			"sign",
			"--ks", opts.Keystore,
			"--ks-pass", opts.KsPass,
			"--ks-key-alias", opts.KeyAlias,
			"--key-pass", opts.KeyPass,
			"--out", outputPath,
		}
		if opts.ExplicitScheme {
			if opts.V1 {
				args = append(args, "--v1-signing-enabled", "true")
			}
			if opts.V2 {
				args = append(args, "--v2-signing-enabled", "true")
			}
			if opts.V3 {
				args = append(args, "--v3-signing-enabled", "true")
			}
			if opts.V4 {
				args = append(args, "--v4-signing-enabled", "true")
			}
		}
		if opts.MinSDK > 0 {
			args = append(args, "--min-sdk-version", fmt.Sprintf("%d", opts.MinSDK))
		}
		args = append(args, apkPath)
		fmt.Println("apksigner", signing.RedactArgs(args))
		return
	}

	schemesMsg := "auto (per APK minSdk)"
	if opts.ExplicitScheme {
		schemesMsg = fmt.Sprintf("v1=%t v2=%t v3=%t v4=%t", opts.V1, opts.V2, opts.V3, opts.V4)
	}
	fmt.Fprintf(os.Stderr, "→ signing: %s\n  keystore: %s\n  alias: %s\n  schemes: %s\n  out: %s\n",
		apkPath, opts.Keystore, opts.KeyAlias, schemesMsg, outputPath)

	if err := signing.Sign(apkPath, outputPath, opts); err != nil {
		fmt.Fprintf(os.Stderr, "wta sign: %v\n", err)
		os.Exit(1)
	}

	if !*noVerify {
		if err := signing.Verify(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "wta sign: verify failed: %v\n", err)
			os.Exit(1)
		}
	}

	st, _ := os.Stat(outputPath)
	fmt.Printf("[OK] signed: %s (%d bytes)\n", outputPath, st.Size())
}
