package kotlin

import (
	"os"
	"regexp"
	"strings"
)

var payloadFuncs = map[string]struct{}{
	"toShellPayload":              {},
	"webViewConfigPayload":        {},
	"floatingWindowConfigPayload": {},
	"errorPageConfigPayload":      {},
	"mediaConfigPayload":          {},
	"htmlConfigPayload":           {},
	"galleryConfigPayload":        {},
	"autoStartConfigPayload":      {},
	"isolationConfigPayload":      {},
	"backgroundRunConfigPayload":  {},
	"notificationConfigPayload":   {},
	"networkTrustConfigPayload":   {},
	"nodejsConfigPayload":         {},
	"pythonAppConfigPayload":      {},
	"goAppConfigPayload":          {},
	"multiWebConfigPayload":       {},
}

var objectReturnPayloadClasses = []string{
	"BackgroundRunConfig",
	"NotificationConfig",
}

var (
	pairKeyRe   = regexp.MustCompile(`"([A-Za-z][A-Za-z0-9_]*)"\s+to\b`)
	funHeaderRe = regexp.MustCompile(`fun\s+ApkConfig\.(\w+)\s*\(\s*\)\s*:`)
)

func ExtractPayloadKeys(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")

	type fnStart struct {
		line int
		name string
	}
	var starts []fnStart
	for i, line := range lines {
		m := funHeaderRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		if !strings.HasSuffix(name, "Payload") {
			continue
		}
		if _, ok := payloadFuncs[name]; !ok {
			continue
		}
		starts = append(starts, fnStart{line: i, name: name})
	}

	out := make(map[string]struct{})
	for i, s := range starts {
		end := len(lines)
		if i+1 < len(starts) {
			end = starts[i+1].line
		}
		body := strings.Join(lines[s.line:end], "\n")
		for _, m := range pairKeyRe.FindAllStringSubmatch(body, -1) {
			out[m[1]] = struct{}{}
		}
	}
	return out, nil
}

func ExtractObjectReturnedKeys(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(data)
	out := make(map[string]struct{})
	valRe := regexp.MustCompile(`(?m)^\s*val\s+(\w+)\s*:`)
	for _, cls := range objectReturnPayloadClasses {
		pattern := regexp.MustCompile(`(?s)data\s+class\s+` + regexp.QuoteMeta(cls) + `\s*\((.*?)\n\)`)
		m := pattern.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		for _, vm := range valRe.FindAllStringSubmatch(m[1], -1) {
			out[vm[1]] = struct{}{}
		}
	}
	return out, nil
}
