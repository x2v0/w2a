package target

import (
	"encoding/json"
	"fmt"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
)

// WebTarget wraps a URL in a WebView shell.
type WebTarget struct{}

func (t *WebTarget) Name() string { return "web" }

func (t *WebTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}
	cfgJSON := map[string]any{
		"appType":   "WEB",
		"language":  "RUSSIAN",
		"targetUrl": cfg.Web.URL,
	}
	if cfg.Web.UserAgent != "" {
		cfgJSON["userAgent"] = cfg.Web.UserAgent
	}
	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	if err := a.AddAssetBytes("assets/app_config.json", data); err != nil {
		return err
	}
	fmt.Printf("web: url=%s\n", cfg.Web.URL)
	return a.SaveAPK(outPath)
}
