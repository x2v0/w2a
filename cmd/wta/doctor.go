package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/target"
)

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // ok | warn | fail | info
	Value  string `json:"value,omitempty"`
	Hint   string `json:"hint,omitempty"`
}

func cmdDoctor(args []string) {
	args, jsonOn := extractJSONFlag(args)

	// simple flags
	asJSON := jsonOn
	for _, a := range args {
		if a == "--json" {
			asJSON = true
		}
	}

	checks := runDoctorChecks()

	if asJSON {
		out := map[string]any{
			"ok":     !hasFail(checks),
			"checks": checks,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		if !out["ok"].(bool) {
			os.Exit(1)
		}
		return
	}

	// text mode
	fmt.Println("wta doctor")
	fmt.Println()
	for _, c := range checks {
		var mark string
		switch c.Status {
		case "ok":
			mark = "✓"
		case "warn":
			mark = "⚠"
		case "fail":
			mark = "✗"
		case "info":
			mark = "ℹ"
		}
		fmt.Printf("  %s %-12s %s\n", mark, c.Name, c.Value)
		if c.Hint != "" {
			fmt.Printf("    %-12s hint: %s\n", "", c.Hint)
		}
	}

	fmt.Println()
	okN, warnN, failN := count(checks)
	fmt.Printf("Status: %d ok, %d warn, %d fail\n", okN, warnN, failN)

	if failN > 0 {
		os.Exit(1)
	}
}

func runDoctorChecks() []doctorCheck {
	var c []doctorCheck

	// 1. go version
	if out, err := exec.Command("go", "version").Output(); err == nil {
		c = append(c, doctorCheck{"go", "ok", strings.TrimSpace(string(out)), ""})
	} else {
		c = append(c, doctorCheck{"go", "fail", "not found", "pkg install golang"})
	}

	// 2. java
	if out, err := exec.Command("java", "-version").CombinedOutput(); err == nil {
		c = append(c, doctorCheck{"java", "ok", firstLine(out), ""})
	} else {
		c = append(c, doctorCheck{"java", "warn", "not found", "needed only for apktool/apksigner (pkg install openjdk-17)"})
	}

	// 3. apksigner
	if path, err := exec.LookPath("apksigner"); err == nil {
		c = append(c, doctorCheck{"apksigner", "ok", path, ""})
	} else {
		c = append(c, doctorCheck{"apksigner", "warn", "not found", "pkg install apksigner (needed for sign step)"})
	}

	// 4. NDK
	if ndk := findNDKForDoctor(); ndk != "" {
		c = append(c, doctorCheck{"ndk", "ok", ndk, ""})
	} else {
		c = append(c, doctorCheck{"ndk", "warn", "not found", "set ANDROID_NDK_HOME (needed for nodejs/go targets)"})
	}

	// 5. rish
	if path, err := exec.LookPath("rish"); err == nil {
		c = append(c, doctorCheck{"rish", "ok", path, ""})
	} else {
		c = append(c, doctorCheck{"rish", "info", "not found", "for install: rish (Shizuku) — optional"})
	}

	// 6. adb
	if path, err := exec.LookPath("adb"); err == nil {
		c = append(c, doctorCheck{"adb", "ok", path, ""})
	} else {
		c = append(c, doctorCheck{"adb", "info", "not found", "optional — pkg install android-tools"})
	}

	// 7. targets
	var impl, stub []string
	allTypes := []string{"web", "html", "frontend", "go", "gallery", "image", "video", "multiweb", "nodejs", "python"}
	for _, t := range allTypes {
		if _, err := target.For(t); err == nil {
			impl = append(impl, t)
		} else {
			stub = append(stub, t)
		}
	}
	val := ""
	if len(impl) > 0 {
		val = "✓ " + strings.Join(impl, "  ✓ ")
	}
	if len(stub) > 0 {
		if val != "" {
			val += "    "
		}
		val += "· " + strings.Join(stub, "  · ")
	}
	c = append(c, doctorCheck{"targets", "info", val, ""})

	// 8. env
	var envParts []string
	for _, k := range []string{"ANDROID_HOME", "ANDROID_NDK_HOME", "ANDROID_NDK_ROOT", "JAVA_HOME", "WTA_TEMPLATE"} {
		v := os.Getenv(k)
		if v == "" {
			envParts = append(envParts, k+"=(unset)")
		} else {
			envParts = append(envParts, k+"="+v)
		}
	}
	c = append(c, doctorCheck{"env", "info", strings.Join(envParts, "  "), ""})

	// 9. default template
	defaultTmpl := os.Getenv("WTA_TEMPLATE")
	if defaultTmpl == "" {
		defaultTmpl = filepath.Join(os.Getenv("HOME"), "build-demo", "webview_shell.apk")
	}
	if st, err := os.Stat(defaultTmpl); err == nil {
		c = append(c, doctorCheck{"template", "ok",
			fmt.Sprintf("%s (%d MB)", defaultTmpl, st.Size()/1024/1024), ""})
	} else {
		c = append(c, doctorCheck{"template", "warn",
			defaultTmpl + " (not found)",
			"pass -template <apk> or set WTA_TEMPLATE"})
	}

	return c
}

// findNDKForDoctor mirrors target.findNDK (unexported there).
// Keep in sync if target.findNDK changes.
func findNDKForDoctor() string {
	candidates := []string{
		os.Getenv("ANDROID_NDK_HOME"),
		os.Getenv("ANDROID_NDK_ROOT"),
	}
	if v := os.Getenv("ANDROID_HOME"); v != "" {
		candidates = append(candidates, filepath.Join(v, "ndk"))
	}
	home := os.Getenv("HOME")
	candidates = append(candidates,
		filepath.Join(home, "android-sdk", "ndk"),
		filepath.Join(home, "Android", "Sdk", "ndk"),
		"/data/data/com.termux/files/usr/lib/android-sdk/ndk",
	)
	for _, base := range candidates {
		if base == "" {
			continue
		}
		if st, err := os.Stat(base); err == nil && st.IsDir() {
			// if this is the ndk root (with source.properties) — return it
			if _, err := os.Stat(filepath.Join(base, "source.properties")); err == nil {
				return base
			}
			// else scan subdirs (versions)
			entries, err := os.ReadDir(base)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				sub := filepath.Join(base, e.Name())
				if _, err := os.Stat(filepath.Join(sub, "source.properties")); err == nil {
					return sub
				}
			}
		}
	}
	return ""
}

func firstLine(b []byte) string {
	s := strings.SplitN(string(b), "\n", 2)[0]
	return strings.TrimSpace(s)
}

func hasFail(checks []doctorCheck) bool {
	for _, c := range checks {
		if c.Status == "fail" {
			return true
		}
	}
	return false
}

func count(checks []doctorCheck) (ok, warn, fail int) {
	for _, c := range checks {
		switch c.Status {
		case "ok":
			ok++
		case "warn":
			warn++
		case "fail":
			fail++
		}
	}
	return
}
