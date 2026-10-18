// Package signing wraps apksigner for APK signing (v1/v2/v3/v4 schemes).
//
// Config sources (priority):
//  1. CLI flags (Options)
//  2. Environment: WTA_KEYSTORE, WTA_KS_PASS, WTA_KEY_ALIAS, WTA_KEY_PASS
//  3. Project wta.yaml `sign:` block
//  4. Global ~/.config/w2a/sign.yaml
package signing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/config"
	"gopkg.in/yaml.v3"
)

// Config mirrors the `sign:` block in wta.yaml and ~/.config/w2a/sign.yaml.
type Config struct {
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

// Options is the fully-resolved set of signing parameters.
type Options struct {
	Keystore       string
	KsPass         string // "pass:xxx" form
	KeyAlias       string
	KeyPass        string // "pass:xxx" form
	V1, V2, V3, V4 bool
	MinSDK         int
	ExplicitScheme bool // true if any V1..V4 were explicitly set
}

// GlobalConfigPath returns ~/.config/w2a/sign.yaml.
func GlobalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "w2a", "sign.yaml")
}

// HasGlobalConfig reports whether ~/.config/w2a/sign.yaml exists.
func HasGlobalConfig() bool {
	_, err := os.Stat(GlobalConfigPath())
	return err == nil
}

// ShouldSign decides whether to sign based on cfg.Sign + presence of
// ~/.config/w2a/sign.yaml.
//
//	sign.Enabled == true  → always sign
//	sign.Enabled == false → never sign
//	sign.Enabled == nil   → auto: sign iff global config exists
func ShouldSign(signSection config.SignSection) bool {
	if signSection.Enabled != nil {
		return *signSection.Enabled
	}
	return HasGlobalConfig()
}

// LoadGlobal reads ~/.config/w2a/sign.yaml (zero Config on absence).
func LoadGlobal() Config {
	data, err := os.ReadFile(GlobalConfigPath())
	if err != nil {
		return Config{}
	}
	var c Config
	_ = yaml.Unmarshal(data, &c)
	return c
}

// LoadProject reads the `sign:` block from a wta.yaml file.
func LoadProject(yamlPath string) Config {
	if yamlPath == "" {
		return Config{}
	}
	cfg, err := config.Load(yamlPath)
	if err != nil {
		return Config{}
	}
	return FromSignSection(cfg.Sign)
}

// FromSignSection converts config.SignSection to signing.Config.
func FromSignSection(s config.SignSection) Config {
	return Config{
		Keystore: s.Keystore, KsPass: s.KsPass,
		KeyAlias: s.KeyAlias, KeyPass: s.KeyPass,
		V1: s.V1, V2: s.V2, V3: s.V3, V4: s.V4,
		MinSDK: s.MinSDK,
	}
}

// Resolve merges priority: CLI → env → project → global.
func Resolve(cli Options, project, global Config) Options {
	pick := func(cliV, envV, projV, globV, defV string) string {
		if cliV != "" {
			return cliV
		}
		if envV != "" {
			return envV
		}
		if projV != "" {
			return projV
		}
		if globV != "" {
			return globV
		}
		return defV
	}

	r := Options{}
	r.Keystore = ExpandHome(pick(cli.Keystore,
		os.Getenv("WTA_KEYSTORE"),
		project.Keystore, global.Keystore,
		"~/.config/w2a/keystore.p12"))
	r.KsPass = NormalizePass(pick(cli.KsPass,
		os.Getenv("WTA_KS_PASS"),
		project.KsPass, global.KsPass, ""))
	r.KeyAlias = pick(cli.KeyAlias,
		os.Getenv("WTA_KEY_ALIAS"),
		project.KeyAlias, global.KeyAlias, "wta")
	r.KeyPass = NormalizePass(pick(cli.KeyPass,
		os.Getenv("WTA_KEY_PASS"),
		project.KeyPass, global.KeyPass, ""))

	r.ExplicitScheme = cli.ExplicitScheme ||
		project.V1 || project.V2 || project.V3 || project.V4 ||
		global.V1 || global.V2 || global.V3 || global.V4
	r.V1 = cli.V1 || project.V1 || global.V1
	r.V2 = cli.V2 || project.V2 || global.V2
	r.V3 = cli.V3 || project.V3 || global.V3
	r.V4 = cli.V4 || project.V4 || global.V4

	r.MinSDK = cli.MinSDK
	if r.MinSDK == 0 {
		r.MinSDK = project.MinSDK
	}
	if r.MinSDK == 0 {
		r.MinSDK = global.MinSDK
	}

	return r
}

// FromConfigAndEnv builds Options from config.SignSection + env + global.
func FromConfigAndEnv(sign config.SignSection) Options {
	return Resolve(Options{}, FromSignSection(sign), LoadGlobal())
}

// Sign signs inPath → outPath via apksigner.
func Sign(inPath, outPath string, o Options) error {
	if _, err := os.Stat(o.Keystore); err != nil {
		return fmt.Errorf("keystore not found: %s", o.Keystore)
	}
	if o.KsPass == "" || o.KeyAlias == "" || o.KeyPass == "" {
		return fmt.Errorf("keystore password / key alias / key password required")
	}

	args := []string{
		"sign",
		"--ks", o.Keystore,
		"--ks-pass", o.KsPass,
		"--ks-key-alias", o.KeyAlias,
		"--key-pass", o.KeyPass,
		"--out", outPath,
	}
	if o.ExplicitScheme {
		if o.V1 {
			args = append(args, "--v1-signing-enabled", "true")
		}
		if o.V2 {
			args = append(args, "--v2-signing-enabled", "true")
		}
		if o.V3 {
			args = append(args, "--v3-signing-enabled", "true")
		}
		if o.V4 {
			args = append(args, "--v4-signing-enabled", "true")
		}
	}
	if o.MinSDK > 0 {
		args = append(args, "--min-sdk-version", fmt.Sprintf("%d", o.MinSDK))
	}
	args = append(args, inPath)

	cmd := exec.Command("apksigner", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apksigner: %w", err)
	}
	return nil
}

// Verify checks an APK's signature via apksigner verify (WARNINGs filtered).
func Verify(path string) error {
	out, err := exec.Command("apksigner", "verify", path).CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "WARNING:") || strings.TrimSpace(line) == "" {
			continue
		}
		fmt.Println(line)
	}
	return err
}

// NormalizePass ensures "pass:xxx" prefix unless already prefixed.
func NormalizePass(p string) string {
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "pass:") || strings.HasPrefix(p, "env:") ||
		strings.HasPrefix(p, "file:") || strings.HasPrefix(p, "stdin") {
		return p
	}
	return "pass:" + p
}

// BoolStr is a convenience for "true"/"false".
func BoolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ExpandHome resolves "~/xxx".
func ExpandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

// RedactArgs replaces "pass:..." with "pass:***" for safe logging.
func RedactArgs(args []string) string {
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
