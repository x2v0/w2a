// Package modules implements the w2a Plugin Market catalog validation
// and submission generation (Go port of scripts/ci/*.py).
package modules

import (
	"fmt"
	"strings"
)

// Report accumulates errors and warnings during a validation run.
type Report struct {
	Errors   []string
	Warnings []string
}

func (r *Report) Error(where, msg string) {
	r.Errors = append(r.Errors, fmt.Sprintf("  ❌ %s: %s", where, msg))
}

func (r *Report) Warning(where, msg string) {
	r.Warnings = append(r.Warnings, fmt.Sprintf("  ⚠️  %s: %s", where, msg))
}

func (r *Report) OK() bool { return len(r.Errors) == 0 }

// Render formats the diagnostics for stdout, mirroring the Python report.
func (r *Report) Render() string {
	var lines []string
	if len(r.Errors) > 0 {
		lines = append(lines, fmt.Sprintf("\n%d error(s):", len(r.Errors)))
		lines = append(lines, r.Errors...)
	}
	if len(r.Warnings) > 0 {
		lines = append(lines, fmt.Sprintf("\n%d warning(s):", len(r.Warnings)))
		lines = append(lines, r.Warnings...)
	}
	if len(lines) == 0 {
		lines = append(lines, "\nAll module checks passed.")
	}
	return strings.Join(lines, "\n")
}
