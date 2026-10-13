package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/x2v0/w2a/go/internal/kotlin"
)

type allowlistFile struct {
	Allowlist []allowlistEntry `json:"allowlist"`
}

type allowlistEntry struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

func main() {
	payloadPath := flag.String("payload", "app/src/main/java/com/webtoapp/core/apkbuilder/ApkConfigJsonFactory.kt", "path to ApkConfigJsonFactory.kt")
	apkconfigPath := flag.String("apkconfig", "app/src/main/java/com/webtoapp/core/apkbuilder/ApkConfig.kt", "path to ApkConfig.kt")
	shellPath := flag.String("shell", "app/src/main/java/com/webtoapp/core/shell/ShellModeManager.kt", "path to ShellModeManager.kt")
	allowlistPath := flag.String("allowlist", "scripts/config_field_drift_allowlist.json", "path to allowlist")
	flag.Parse()

	payloadKeys, err := kotlin.ExtractPayloadKeys(*payloadPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "payload parse error: %v\n", err)
		os.Exit(1)
	}
	objKeys, err := kotlin.ExtractObjectReturnedKeys(*apkconfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "apkconfig parse error: %v\n", err)
		os.Exit(1)
	}
	for k := range objKeys {
		payloadKeys[k] = struct{}{}
	}

	shellKeys, err := kotlin.ExtractSerializedNames(*shellPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shell parse error: %v\n", err)
		os.Exit(1)
	}

	var allow allowlistFile
	if data, err := os.ReadFile(*allowlistPath); err == nil {
		if err := json.Unmarshal(data, &allow); err != nil {
			fmt.Fprintf(os.Stderr, "allowlist parse error: %v\n", err)
			os.Exit(1)
		}
	}
	allowKeys := make(map[string]struct{}, len(allow.Allowlist))
	allowReasons := make(map[string]string, len(allow.Allowlist))
	for _, e := range allow.Allowlist {
		if e.Key == "" {
			continue
		}
		allowKeys[e.Key] = struct{}{}
		allowReasons[e.Key] = strings.TrimSpace(e.Reason)
	}

	var errors []string
	var warnings []string

	var noReason []string
	for k := range allowKeys {
		if allowReasons[k] == "" {
			noReason = append(noReason, k)
		}
	}
	sort.Strings(noReason)
	if len(noReason) > 0 {
		errors = append(errors, "allowlist entries missing 'reason': "+strings.Join(noReason, ", "))
	}

	var stale []string
	for k := range allowKeys {
		if _, ok := payloadKeys[k]; ok {
			continue
		}
		if _, ok := shellKeys[k]; ok {
			continue
		}
		stale = append(stale, k)
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		warnings = append(warnings, "allowlist contains keys present in neither side (stale): "+strings.Join(stale, ", "))
	}

	var onlyPayload, onlyShell []string
	for k := range payloadKeys {
		if _, ok := shellKeys[k]; ok {
			continue
		}
		if _, ok := allowKeys[k]; ok {
			continue
		}
		onlyPayload = append(onlyPayload, k)
	}
	for k := range shellKeys {
		if _, ok := payloadKeys[k]; ok {
			continue
		}
		if _, ok := allowKeys[k]; ok {
			continue
		}
		onlyShell = append(onlyShell, k)
	}
	sort.Strings(onlyPayload)
	sort.Strings(onlyShell)

	for _, k := range onlyPayload {
		errors = append(errors, fmt.Sprintf("payload writes key '%s' but shell has no @SerializedName", k))
	}
	for _, k := range onlyShell {
		errors = append(errors, fmt.Sprintf("shell declares @SerializedName('%s') but payload never writes it", k))
	}

	for _, w := range warnings {
		fmt.Printf("WARN: %s\n", w)
	}
	if len(errors) > 0 {
		for _, e := range errors {
			fmt.Printf("ERROR: %s\n", e)
		}
		fmt.Printf("check_config_field_drift: FAILED (payload_only=%d shell_only=%d allowlist=%d)\n",
			len(onlyPayload), len(onlyShell), len(allowKeys))
		os.Exit(1)
	}

	shared := 0
	for k := range payloadKeys {
		if _, ok := shellKeys[k]; ok {
			shared++
		}
	}
	fmt.Printf("check_config_field_drift: OK (payload=%d shell=%d allowlist=%d shared=%d)\n",
		len(payloadKeys), len(shellKeys), len(allowKeys), shared)
}
