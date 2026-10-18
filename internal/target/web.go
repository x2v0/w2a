package target

import (
	"encoding/json"
	"fmt"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
)

// WebTarget wraps a URL in a WebView shell.
//
// Output matches WebToApp's `ApkConfigJsonFactory.toShellPayload()`:
// flat map with schemaVersion, appType, targetUrl, plus nested
// webViewConfig object. Only explicitly-set fields are emitted — shell
// defaults apply for everything else.
type WebTarget struct{}

func (t *WebTarget) Name() string { return "web" }

func (t *WebTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}

	w := cfg.Web
	if w == nil {
		return fmt.Errorf("web: nil WebSection")
	}

	cfgJSON := map[string]any{
		"schemaVersion": 1,
		"appType":       "WEB",
		"targetUrl":     w.URL,
		"language":      pickString(cfg.App.Language, "RUSSIAN"),
	}

	// legacy top-level (old shells) — harmless if ignored
	if w.UserAgent != "" {
		cfgJSON["userAgent"] = w.UserAgent
	}

	// --- webViewConfig (canonical, matches Kotlin WebViewBlock + WebViewBehaviorBlock) ---
	webViewCfg := map[string]any{}

	putBool := func(key string, v *bool) {
		if v != nil {
			webViewCfg[key] = *v
		}
	}
	putString := func(key, v string) {
		if v != "" {
			webViewCfg[key] = v
		}
	}
	putInt := func(key string, v int) {
		if v != 0 {
			webViewCfg[key] = v
		}
	}

	// userAgent — combine legacy + custom
	if w.UserAgent != "" && w.CustomUserAgent == "" {
		webViewCfg["userAgent"] = w.UserAgent
	}
	putString("userAgentMode", w.UserAgentMode)
	putString("customUserAgent", w.CustomUserAgent)

	// WebViewBlock bools
	putBool("javaScriptEnabled", w.JavaScriptEnabled)
	putBool("domStorageEnabled", w.DomStorageEnabled)
	putBool("allowFileAccess", w.AllowFileAccess)
	putBool("allowContentAccess", w.AllowContentAccess)
	putBool("cacheEnabled", w.CacheEnabled)
	putBool("clearBrowsingDataOnLaunch", w.ClearBrowsingDataOnLaunch)
	putBool("zoomEnabled", w.ZoomEnabled)
	putBool("desktopMode", w.DesktopMode)
	putBool("swipeRefreshEnabled", w.SwipeRefreshEnabled)
	putBool("overscrollEffectEnabled", w.OverscrollEffectEnabled)
	putBool("hideToolbar", w.HideToolbar)
	putBool("browserToolbarEnabled", w.BrowserToolbarEnabled)

	// WebViewBlock strings
	putString("orientationMode", w.OrientationMode)
	putString("keyboardAdjustMode", w.KeyboardAdjustMode)

	// WebViewBehaviorBlock
	putInt("initialScale", w.InitialScale)
	putInt("pageZoomPercent", w.PageZoomPercent)
	putString("viewportMode", w.ViewportMode)
	putString("newWindowBehavior", w.NewWindowBehavior)
	putBool("enablePaymentSchemes", w.EnablePaymentSchemes)
	putBool("enableShareBridge", w.EnableShareBridge)
	putBool("mediaAutoplayEnabled", w.MediaAutoplayEnabled)
	putBool("acceptThirdPartyCookies", w.AcceptThirdPartyCookies)
	putBool("javaScriptCanOpenWindows", w.JavaScriptCanOpenWindows)

	if len(webViewCfg) > 0 {
		cfgJSON["webViewConfig"] = webViewCfg
	}

	// --- Meta ---
	putMetaString(cfgJSON, "themeType", w.ThemeType)
	putMetaString(cfgJSON, "darkMode", w.DarkMode)
	putMetaString(cfgJSON, "engineType", w.Engine)

	// --- Flat format (for shells/go-webview minimal Java shell) ---
	cfgJSON["url"] = w.URL // alias for targetUrl
	if w.UserAgent != "" {
		cfgJSON["userAgent"] = w.UserAgent
	}
	cfgJSON["javaScriptEnabled"] = true
	if w.JavaScriptEnabled != nil {
		cfgJSON["javaScriptEnabled"] = *w.JavaScriptEnabled
	}
	cfgJSON["domStorageEnabled"] = true
	if w.DomStorageEnabled != nil {
		cfgJSON["domStorageEnabled"] = *w.DomStorageEnabled
	}
	cfgJSON["zoomEnabled"] = false
	if w.ZoomEnabled != nil {
		cfgJSON["zoomEnabled"] = *w.ZoomEnabled
	}
	cfgJSON["hideStatusBar"] = false
	if w.HideStatusBar != nil {
		cfgJSON["hideStatusBar"] = *w.HideStatusBar
	}
	if w.ClearBrowsingDataOnLaunch != nil {
		cfgJSON["clearCacheOnLaunch"] = *w.ClearBrowsingDataOnLaunch
	}
	if w.OrientationMode != "" {
		cfgJSON["orientation"] = w.OrientationMode
	} else {
		cfgJSON["orientation"] = "auto"
	}
	if w.DarkMode != "" {
		cfgJSON["darkMode"] = w.DarkMode
	} else {
		cfgJSON["darkMode"] = "system"
	}
	if w.InjectCss != "" {
		cfgJSON["injectCss"] = w.InjectCss
	}
	if w.InjectJs != "" {
		cfgJSON["injectJs"] = w.InjectJs
	}
	if w.ExternalLinks != "" {
		cfgJSON["externalLinks"] = w.ExternalLinks
	} else {
		cfgJSON["externalLinks"] = "browser"
	}
	if cfg.Splash.Enabled || cfg.Splash.Color != "" || cfg.Splash.Duration > 0 {
		splashDur := cfg.Splash.Duration
		if splashDur == 0 {
			splashDur = 1000
		}
		color := cfg.Splash.Color
		if color == "" {
			color = "#1a1a1a"
		}
		cfgJSON["splash"] = map[string]any{
			"enabled":  cfg.Splash.Enabled,
			"color":    color,
			"duration": splashDur,
		}
	}

	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	if err := a.AddAssetBytes("assets/app_config.json", data); err != nil {
		return err
	}

	// Log summary
	fields := 0
	if len(webViewCfg) > 0 {
		fields = len(webViewCfg)
	}
	fmt.Printf("web: url=%s webViewConfig=%d fields\n", w.URL, fields)
	return a.SaveAPK(outPath)
}

func pickString(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func putMetaString(m map[string]any, key, v string) {
	if v != "" {
		m[key] = v
	}
}
