// Command wta is the WebToApp CLI — builds APKs from a wta.yaml config.
package main

import (
	"flag"
	"fmt"
	"os"

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
	fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		die(err)
	}
	if *verbose {
		fmt.Printf("config: %s\ntype:   %s\noutput: %s\n", *cfgPath, cfg.Type, cfg.Resolve(cfg.Output.Path))
	}

	res, err := build.Build(cfg)
	if err != nil {
		die(err)
	}
	fmt.Printf("\nbuilt: %s (%d bytes)\n", res.OutputPath, res.Size)
	fmt.Printf("stages: %v\n", res.Targets)
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
