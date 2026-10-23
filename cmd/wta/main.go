// Command wta is the w2a CLI — builds APKs from a wta.yaml config.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/x2v0/w2a/go/internal/build"
	"github.com/x2v0/w2a/go/internal/config"
)

const usage = `wta — w2a CLI

Usage:
  wta build    -c wta.yaml [--json]     Build APK from config
  wta sign <apk>                         Sign APK (apksigner wrapper)
wta validate -c wta.yaml              Validate config only
  wta inspect  <apk> [-v] [--json]      Inspect built APK
  wta types                             List supported app types
  wta test     <dir> [--json] [--filter <s>] [--list]  Build/list *.yaml
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
	case "test":
		cmdTest(os.Args[2:])
	case "doctor":
		cmdDoctor(os.Args[2:])
	case "sign":
		cmdSign(os.Args[2:])
	case "verify":
		cmdVerify(os.Args[2:])
	case "version", "-v", "--version":
		cmdVersion(os.Args[2:])
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "wta: unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func cmdBuild(args []string) {
	args, jsonOn := extractJSONFlag(args)

	fs := flag.NewFlagSet("build", flag.ExitOnError)
	cfgPath := fs.String("c", "wta.yaml", "config file")
	verbose := fs.Bool("v", false, "verbose")
	asJSON := fs.Bool("json", jsonOn, "machine-readable JSON output")
	outOverride := fs.String("out", "", "override output.path from YAML")
	templateOverride := fs.String("template", "", "override shell template path")
	signFlag := fs.Bool("sign", false, "force signing (even if sign: block absent)")
	noSignFlag := fs.Bool("no-sign", false, "force unsigned output")
	fs.Parse(args)

	if *signFlag && *noSignFlag {
		die(fmt.Errorf("--sign and --no-sign are mutually exclusive"))
	}

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
	// CLI overrides for sign decision (nil = auto via sign.yaml)
	if *signFlag {
		enabled := true
		cfg.Sign.Enabled = &enabled
	} else if *noSignFlag {
		enabled := false
		cfg.Sign.Enabled = &enabled
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
// Структура совместима с validate --json: {ok, errors, error}.
func jsonError(err error) {
	out := map[string]any{
		"ok":     false,
		"errors": 1,
		"error":  err.Error(),
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
	args, jsonOn := extractJSONFlag(args)

	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	cfgPath := fs.String("c", "wta.yaml", "config file")
	asJSON := fs.Bool("json", jsonOn, "machine-readable JSON output")
	fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		if *asJSON {
			jsonError(err)
		}
		die(err)
	}

	type check struct {
		Name  string `json:"name"`
		OK    bool   `json:"ok"`
		Value string `json:"value,omitempty"`
		Error string `json:"error,omitempty"`
	}
	var checks []check
	add := func(name string, ok bool, value, errMsg string) {
		checks = append(checks, check{Name: name, OK: ok, Value: value, Error: errMsg})
	}

	// type
	knownTypes := map[string]bool{
		"web": true, "html": true, "frontend": true, "go": true,
		"gallery": true, "image": true, "video": true, "multiweb": true,
		"nodejs": true, "python": true,
	}
	if cfg.Type == "" {
		add("type", false, "", "empty")
	} else if !knownTypes[cfg.Type] {
		add("type", false, cfg.Type, "unknown type")
	} else {
		add("type", true, cfg.Type, "")
	}

	// package
	pkgRe := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)+$`)
	if cfg.App.Package == "" {
		add("package", false, "", "empty")
	} else if !pkgRe.MatchString(cfg.App.Package) {
		add("package", false, cfg.App.Package, "invalid format")
	} else {
		add("package", true, cfg.App.Package, "")
	}

	// versionCode
	if cfg.App.VersionCode == 0 {
		add("versionCode", false, "0", "should be > 0")
	} else {
		add("versionCode", true, fmt.Sprintf("%d", cfg.App.VersionCode), "")
	}

	// versionName
	if cfg.App.VersionName == "" {
		add("versionName", false, "", "empty")
	} else {
		add("versionName", true, cfg.App.VersionName, "")
	}

	// template
	tmplPath := cfg.Template.APK
	if tmplPath == "" {
		tmplPath = os.Getenv("WTA_TEMPLATE")
	}
	if tmplPath == "" {
		tmplPath = "app/src/main/assets/template/webview_shell.apk"
	}
	tmplPath = cfg.Resolve(tmplPath)
	if _, err := os.Stat(tmplPath); err != nil {
		add("template", false, tmplPath, "not found")
	} else {
		add("template", true, tmplPath, "")
	}

	// output
	if cfg.Output.Path == "" {
		add("output", false, "", "empty")
	} else {
		add("output", true, cfg.Resolve(cfg.Output.Path), "")
	}

	// output-in-source warning (advisory, does not fail validation)
	if w := validateSourceOutput(cfg); w != "" {
		add("output-in-source", true, "⚠ "+w, "")
	}

	// source (для типов с локальными файлами)
	sourceCheck := func(kind, src string) {
		if src == "" {
			add("source", false, "", "empty ("+kind+")")
			return
		}
		full := cfg.Resolve(src)
		if _, err := os.Stat(full); err != nil {
			add("source", false, full, "not found ("+kind+")")
		} else {
			add("source", true, full, "")
		}
	}
	switch cfg.Type {
	case "html":
		if cfg.HTML != nil {
			sourceCheck("html", cfg.HTML.Source)
		}
	case "frontend":
		if cfg.Frontend != nil {
			sourceCheck("frontend", cfg.Frontend.Source)
		}
	case "go":
		if cfg.Go != nil {
			sourceCheck("go", cfg.Go.Source)
		}
	case "gallery":
		if cfg.Gallery != nil {
			sourceCheck("gallery", cfg.Gallery.Source)
		}
	case "image":
		if cfg.Image != nil {
			sourceCheck("image", cfg.Image.Source)
		}
	case "video":
		if cfg.Video != nil {
			sourceCheck("video", cfg.Video.Source)
		}
	case "web":
		if cfg.Web == nil || cfg.Web.URL == "" {
			add("web.url", false, "", "empty")
		} else {
			add("web.url", true, cfg.Web.URL, "")
		}
	case "multiweb":
		if cfg.MultiWeb == nil || len(cfg.MultiWeb.Sites) == 0 {
			add("multiweb.sites", false, "0", "at least 1 site required")
		} else {
			add("multiweb.sites", true, fmt.Sprintf("%d sites", len(cfg.MultiWeb.Sites)), "")
		}
	}

	// permissions
	add("permissions", true,
		fmt.Sprintf("%d add, %d remove", len(cfg.Permissions.Add), len(cfg.Permissions.Remove)), "")

	// подсчёт ошибок
	errors := 0
	for _, c := range checks {
		if !c.OK {
			errors++
		}
	}

	if *asJSON {
		out := map[string]any{
			"ok":     errors == 0,
			"errors": errors,
			"checks": checks,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		if errors > 0 {
			os.Exit(1)
		}
		return
	}

	for _, c := range checks {
		mark := "✓"
		if !c.OK {
			mark = "✗"
		}
		if c.OK {
			fmt.Printf("%s %-16s %s\n", mark, c.Name+":", c.Value)
		} else {
			fmt.Printf("%s %-16s %s\n", mark, c.Name+":", c.Error)
		}
	}
	fmt.Println()
	if errors == 0 {
		fmt.Println("✅ OK")
	} else {
		fmt.Printf("❌ FAIL: %d error(s)\n", errors)
		os.Exit(1)
	}
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	typ := fs.String("type", "web", "app type")
	name := fs.String("name", "My App", "app name")
	pkg := fs.String("package", "com.example.app", "package name")
	url := fs.String("url", "https://example.com", "target URL (for type=web)")
	withDebug := fs.Bool("with-debug", false, "scaffold ./html-project/ with mtapi-debug.js (only for -type html)")
	fs.Parse(args)

	tmpl := starterYAML(*typ, *name, *pkg, *url)
	fmt.Print(tmpl)

	switch *typ {
	case "html":
		if *withDebug {
			if err := scaffoldHTMLProject(*name, true); err != nil {
				die(err)
			}
			fmt.Fprintln(os.Stderr, "  scaffolded ./html-project/ (index.html + mtapi-debug.js)")
		}
	case "chat":
		if err := scaffoldChat(*name); err != nil {
			die(err)
		}
		fmt.Fprintln(os.Stderr, "  scaffolded ./html-project/ (chat UI + mtapi-debug.js)")
	case "notes":
		if err := scaffoldNotes(*name); err != nil {
			die(err)
		}
		fmt.Fprintln(os.Stderr, "  scaffolded ./html-project/ (notes UI + mtapi-debug.js)")
	case "dashboard":
		if err := scaffoldDashboard(*name); err != nil {
			die(err)
		}
		fmt.Fprintln(os.Stderr, "  scaffolded ./html-project/ (dashboard UI + mtapi-debug.js)")
	case "rss":
		if err := scaffoldRss(*name); err != nil {
			die(err)
		}
		fmt.Fprintln(os.Stderr, "  scaffolded ./html-project/ (RSS reader + mtapi-debug.js)")
	case "reader":
		if err := scaffoldReader(*name); err != nil {
			die(err)
		}
		fmt.Fprintln(os.Stderr, "  scaffolded ./html-project/ (reader MD/HTML/TXT + mtapi-debug.js)")
	}
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "wta: %v\n", err)
	os.Exit(1)
}

// extractJSONFlag removes --json / -json from args (anywhere, not just before
// positional args) and returns the filtered list + whether the flag was set.
// Needed because Go's flag package stops parsing at the first positional arg,
// so "wta test dir --json" would silently ignore --json.
func extractJSONFlag(args []string) ([]string, bool) {
	var out []string
	found := false
	for _, a := range args {
		if a == "--json" || a == "-json" || a == "--json=true" {
			found = true
			continue
		}
		out = append(out, a)
	}
	return out, found
}

// extractTestFlags — pre-scan для cmdTest.
// Go flag.Parse останавливается на первом positional, поэтому:
//
//	wta test dir --filter go    ← --filter не парсится
//	wta test dir --json         ← --json не парсится
//
// Убираем их из args вручную.
func extractTestFlags(args []string) (filtered []string, jsonOn bool, filter string, listOn bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--json" || a == "-json" {
			jsonOn = true
			continue
		}
		if a == "--list" || a == "-list" {
			listOn = true
			continue
		}
		if a == "--filter" || a == "-filter" {
			if i+1 < len(args) {
				filter = args[i+1]
				i++
			}
			continue
		}
		if strings.HasPrefix(a, "--filter=") {
			filter = a[len("--filter="):]
			continue
		}
		if strings.HasPrefix(a, "-filter=") {
			filter = a[len("-filter="):]
			continue
		}
		filtered = append(filtered, a)
	}
	return
}
