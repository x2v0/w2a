// Command wta is the WebToApp CLI — builds APKs from a wta.yaml config.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/x2v0/w2a/go/internal/build"
	"github.com/x2v0/w2a/go/internal/config"
)

const usage = `wta — WebToApp CLI

Usage:
  wta build    -c wta.yaml [--json]     Build APK from config
  wta validate -c wta.yaml              Validate config only
  wta inspect  <apk> [-v] [--json]      Inspect built APK
  wta types                             List supported app types
  wta init     -type web -name "..."    Generate starter wta.yaml
  wta version                           Print version

Examples:
  wta build -c examples/web.yaml
  wta build -c wta.yaml --json
  wta validate -c examples/html.yaml
  wta inspect /sdcard/Download/app.apk
  wta init -type web -name "My App" -package com.me.app > wta.yaml
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "build":
		cmdBuild(os.Args[2:])
	case "validate":
		cmdValidate(os.Args[2:])
	case "init":
		cmdInit(os.Args[2:])
	case "inspect":
		cmdInspect(os.Args[2:])
	case "types":
		cmdTypes(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println("wta 0.1.0")
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "wta: unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func cmdBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	cfgPath := fs.String("c", "wta.yaml", "config file")
	verbose := fs.Bool("v", false, "verbose")
	asJSON := fs.Bool("json", false, "machine-readable JSON output")
	outOverride := fs.String("out", "", "override output.path from YAML")
	templateOverride := fs.String("template", "", "override shell template path")
	fs.Parse(args)

	t0 := time.Now()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		if *asJSON {
			jsonError(err)
		}
		die(err)
	}

	if *outOverride != "" {
		cfg.Output.Path = *outOverride
	}
	if *templateOverride != "" {
		cfg.Template.APK = *templateOverride
	}

	if *verbose && !*asJSON {
		fmt.Printf("config: %s\ntype:   %s\noutput: %s\n",
			*cfgPath, cfg.Type, cfg.Resolve(cfg.Output.Path))
	}

	// В --json режиме заглушаем stdout build.Build (techlogs → stderr),
	// чтобы JSON был единственным выводом в stdout.
	var res *build.Result
	var buildErr error
	if *asJSON {
		realStdout := os.Stdout
		os.Stdout = os.Stderr
		res, buildErr = build.Build(cfg)
		os.Stdout = realStdout
	} else {
		res, buildErr = build.Build(cfg)
	}
	if buildErr != nil {
		if *asJSON {
			jsonError(buildErr)
		}
		die(buildErr)
	}

	dur := time.Since(t0).Milliseconds()

	if *asJSON {
		out := map[string]any{
			"ok":         true,
			"config":     *cfgPath,
			"type":       cfg.Type,
			"package":    cfg.App.Package,
			"path":       res.OutputPath,
			"size":       res.Size,
			"stages":     res.Targets,
			"durationMs": dur,
		}
		if sum, err := fileSHA256(res.OutputPath); err == nil {
			out["sha256"] = sum
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return
	}

	fmt.Printf("\nbuilt: %s (%d bytes)\n", res.OutputPath, res.Size)
	fmt.Printf("stages: %v\n", res.Targets)
	fmt.Printf("time:   %d ms\n", dur)
}

// jsonError печатает ошибку в JSON-формате и выходит с кодом 1.
func jsonError(err error) {
	out := map[string]any{
		"ok":    false,
		"error": err.Error(),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
	os.Exit(1)
}

// fileSHA256 считает sha256 файла.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func cmdValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	cfgPath := fs.String("c", "wta.yaml", "config file")
	fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		die(err)
	}
	fmt.Printf("OK: type=%s package=%s\n", cfg.Type, cfg.App.Package)
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	typ := fs.String("type", "web", "app type")
	name := fs.String("name", "My App", "app name")
	pkg := fs.String("package", "com.example.app", "package name")
	url := fs.String("url", "https://example.com", "target URL (for type=web)")
	fs.Parse(args)

	tmpl := starterYAML(*typ, *name, *pkg, *url)
	fmt.Print(tmpl)
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "wta: %v\n", err)
	os.Exit(1)
}
