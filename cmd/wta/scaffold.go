package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed mtapi-debug.js
var mtapiDebugJS string

const htmlProjectDir = "html-project"

// scaffoldHTMLProject создаёт ./html-project/ с index.html (+ mtapi-debug.js если withDebug)
func scaffoldHTMLProject(appName string, withDebug bool) error {
	if err := os.MkdirAll(htmlProjectDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", htmlProjectDir, err)
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"utf-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	sb.WriteString("  <title>" + appName + "</title>\n")
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: system-ui, sans-serif; padding: 16px; }\n")
	sb.WriteString("    button { padding: 12px 20px; font-size: 16px; margin: 8px; }\n")
	sb.WriteString("  </style>\n</head>\n<body>\n")
	sb.WriteString("  <h1>" + appName + "</h1>\n")
	sb.WriteString("  <button onclick=\"console.log('Hello from " + appName + "!')\">Test</button>\n")
	sb.WriteString("  <button onclick=\"alert(document.title)\">Title</button>\n")
	if withDebug {
		sb.WriteString("\n  <!-- mtapi-debug: zero-dep mobile DevTools -->\n")
		sb.WriteString("  <script src=\"mtapi-debug.js\"></script>\n")
		sb.WriteString("  <script>MtapiDebug.attach({ container: document.body });</script>\n")
	}
	sb.WriteString("</body>\n</html>\n")

	if err := os.WriteFile(filepath.Join(htmlProjectDir, "index.html"), []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write index.html: %w", err)
	}

	if withDebug {
		if err := os.WriteFile(filepath.Join(htmlProjectDir, "mtapi-debug.js"), []byte(mtapiDebugJS), 0644); err != nil {
			return fmt.Errorf("write mtapi-debug.js: %w", err)
		}
	}
	return nil
}
