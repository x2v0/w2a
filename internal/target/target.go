// Package target implements per-type APK preparation (Web, HTML, Go, ...).
package target

import (
	"fmt"

	"github.com/x2v0/w2a/go/internal/config"
)

// Target prepares a template APK for a specific app type.
type Target interface {
	// Name returns the target identifier ("web", "html", ...).
	Name() string

	// Prepare takes the input template APK and writes a modified APK to
	// outPath. It typically injects assets and config.json, but does NOT
	// apply common mods (package name, permissions) — those run after.
	Prepare(cfg *config.Config, templatePath, outPath string) error
}

// For returns the Target implementation for a config type.
func For(appType string) (Target, error) {
	switch appType {
	case "web":
		return &WebTarget{}, nil
	case "html":
		return &HTMLTarget{}, nil
	case "frontend":
		return &FrontendTarget{}, nil
	case "go":
		return &GoTarget{}, nil
	case "gallery":
		return &GalleryTarget{}, nil
	case "image":
		return &ImageTarget{}, nil
	case "video":
		return &VideoTarget{}, nil
	case "multiweb":
		return &MultiWebTarget{}, nil
	case "nodejs":
		return &NodeJSTarget{}, nil
	case "php", "wordpress":
		return nil, fmt.Errorf("target %q was removed in 2026-10 (PHP/WordPress unsupported)", appType)
	case "python":
		return nil, fmt.Errorf("target %q not yet implemented in Go (use Kotlin GUI)", appType)
	}
	return nil, fmt.Errorf("unknown target %q", appType)
}
