package target

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
)

// MultiWebTarget bundles several websites into one APK with a switcher UI.
type MultiWebTarget struct{}

func (t *MultiWebTarget) Name() string { return "multiweb" }

func (t *MultiWebTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}
	type siteOut struct {
		Name string `json:"name"`
		URL  string `json:"url"`
		Icon string `json:"icon,omitempty"`
	}
	var sites []siteOut
	for _, s := range cfg.MultiWeb.Sites {
		out := siteOut{Name: s.Name, URL: s.URL}
		if s.Icon != "" {
			iconPath := cfg.Resolve(s.Icon)
			data, err := os.ReadFile(iconPath)
			if err == nil {
				dst := "assets/multiweb/icons/" + s.Name + ".png"
				_ = a.AddAssetBytes(dst, data)
				out.Icon = dst
			}
		}
		sites = append(sites, out)
	}
	// Shell reads multiWebConfig.sites (camelCase W in JSON key).
	// MultiWebSiteShellConfig: {id, name, url, type, localFilePath}.
	enriched := make([]map[string]any, 0, len(sites))
	for i, site := range sites {
		name := site.Name
		if name == "" {
			name = site.URL
		}
		enriched = append(enriched, map[string]any{
			"id":            fmt.Sprintf("site-%d", i),
			"name":          name,
			"url":           site.URL,
			"type":          "URL",
			"localFilePath": "",
		})
	}

	cfgJSON := map[string]any{
		"appType":  "MULTI_WEB",
		"language": "RUSSIAN",
		"multiWebConfig": map[string]any{
			"sites":           enriched,
			"displayMode":     "TABS",
			"refreshInterval": 30,
			"showSiteIcons":   true,
			"cardColumns":     2,
			"drawerColumns":   1,
		},
	}
	body, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/app_config.json", body)
	fmt.Printf("multiweb: %d sites\n", len(sites))
	return a.SaveAPK(outPath)
}
