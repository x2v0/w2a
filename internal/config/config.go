// Package config loads and validates wta.yaml build specifications.
package config

// Config is the top-level build specification.
type Config struct {
	App         AppSection         `yaml:"app"`
	Type        string             `yaml:"type"`
	Web         *WebSection        `yaml:"web,omitempty"`
	HTML        *HTMLSection       `yaml:"html,omitempty"`
	Frontend    *FrontendSection   `yaml:"frontend,omitempty"`
	NodeJS      *NodeJSSection     `yaml:"nodejs,omitempty"`
	Go          *GoSection         `yaml:"go,omitempty"`
	Python      *PythonSection     `yaml:"python,omitempty"`
	PHP         *PHPSection        `yaml:"php,omitempty"`
	WordPress   *WordPressSection  `yaml:"wordpress,omitempty"`
	Gallery     *GallerySection    `yaml:"gallery,omitempty"`
	Image       *ImageSection      `yaml:"image,omitempty"`
	Video       *VideoSection      `yaml:"video,omitempty"`
	MultiWeb    *MultiWebSection   `yaml:"multiweb,omitempty"`
	Permissions PermissionsSection `yaml:"permissions,omitempty"`
	Locales     LocalesSection     `yaml:"locales,omitempty"`
	Features    FeaturesSection    `yaml:"features,omitempty"`
	Template    TemplateSection    `yaml:"template,omitempty"`
	Sign        SignSection        `yaml:"sign,omitempty"`
	Output      OutputSection      `yaml:"output"`

	BaseDir string `yaml:"-"` // directory of the config file
}

type AppSection struct {
	Name        string `yaml:"name"`
	Package     string `yaml:"package"`
	VersionCode uint32 `yaml:"versionCode"`
	VersionName string `yaml:"versionName"`
	Icon        string `yaml:"icon"`
	Engine      string `yaml:"engine"`   // webview | geckoview
	Language    string `yaml:"language"` // RUSSIAN | ENGLISH | CHINESE | ... (AppLanguage enum)
}

type WebSection struct {
	URL       string `yaml:"url"`
	UserAgent string `yaml:"userAgent"`

	// --- WebView block (mirror of Kotlin WebViewBlock) ---
	// Pointers: nil = shell default (do not override).
	JavaScriptEnabled         *bool  `yaml:"javaScriptEnabled,omitempty"`
	DomStorageEnabled         *bool  `yaml:"domStorageEnabled,omitempty"`
	AllowFileAccess           *bool  `yaml:"allowFileAccess,omitempty"`
	AllowContentAccess        *bool  `yaml:"allowContentAccess,omitempty"`
	CacheEnabled              *bool  `yaml:"cacheEnabled,omitempty"`
	ClearBrowsingDataOnLaunch *bool  `yaml:"clearBrowsingDataOnLaunch,omitempty"`
	ZoomEnabled               *bool  `yaml:"zoomEnabled,omitempty"`
	DesktopMode               *bool  `yaml:"desktopMode,omitempty"`
	SwipeRefreshEnabled       *bool  `yaml:"swipeRefreshEnabled,omitempty"`
	OverscrollEffectEnabled   *bool  `yaml:"overscrollEffectEnabled,omitempty"`
	HideToolbar               *bool  `yaml:"hideToolbar,omitempty"`
	BrowserToolbarEnabled     *bool  `yaml:"browserToolbarEnabled,omitempty"`
	UserAgentMode             string `yaml:"userAgentMode,omitempty"` // DEFAULT|DESKTOP|CUSTOM
	CustomUserAgent           string `yaml:"customUserAgent,omitempty"`
	OrientationMode           string `yaml:"orientationMode,omitempty"` // AUTO|PORTRAIT|LANDSCAPE
	KeyboardAdjustMode        string `yaml:"keyboardAdjustMode,omitempty"`

	// --- WebView behavior (mirror of Kotlin WebViewBehaviorBlock) ---
	InitialScale             int    `yaml:"initialScale,omitempty"`
	PageZoomPercent          int    `yaml:"pageZoomPercent,omitempty"`
	ViewportMode             string `yaml:"viewportMode,omitempty"`
	NewWindowBehavior        string `yaml:"newWindowBehavior,omitempty"` // SAME_WINDOW|NEW_WINDOW
	EnablePaymentSchemes     *bool  `yaml:"enablePaymentSchemes,omitempty"`
	EnableShareBridge        *bool  `yaml:"enableShareBridge,omitempty"`
	MediaAutoplayEnabled     *bool  `yaml:"mediaAutoplayEnabled,omitempty"`
	AcceptThirdPartyCookies  *bool  `yaml:"acceptThirdPartyCookies,omitempty"`
	JavaScriptCanOpenWindows *bool  `yaml:"javaScriptCanOpenWindows,omitempty"`

	// --- Meta block (mirror of Kotlin MetaBlock, app-level) ---
	ThemeType string `yaml:"themeType,omitempty"` // AURORA|...
	DarkMode  string `yaml:"darkMode,omitempty"`  // SYSTEM|LIGHT|DARK
	Engine    string `yaml:"engine,omitempty"`    // SYSTEM_WEBVIEW|GECKOVIEW
}

type HTMLSection struct {
	Source    string `yaml:"source"`
	EntryFile string `yaml:"entryFile"`
}

type FrontendSection struct {
	Source string `yaml:"source"`
}

type NodeJSSection struct {
	Source      string            `yaml:"source"`
	EntryModule string            `yaml:"entryModule"`
	Port        int               `yaml:"port"`
	NodeVersion string            `yaml:"nodeVersion"`
	ABIs        []string          `yaml:"abis"`
	EnvVars     map[string]string `yaml:"envVars,omitempty"`
}

type GoSection struct {
	Source string   `yaml:"source"`
	ABIs   []string `yaml:"abis"`
	NoCGO  bool     `yaml:"noCgo"`
}

type PythonSection struct {
	Source      string `yaml:"source"`
	EntryModule string `yaml:"entryModule"`
}

type PHPSection struct {
	Source       string `yaml:"source"`
	DocumentRoot string `yaml:"documentRoot"`
}

type WordPressSection struct {
	Source  string `yaml:"source"`
	SiteURL string `yaml:"siteUrl"`
}

type GallerySection struct {
	Source  string `yaml:"source"`
	Columns int    `yaml:"columns"`
}

type ImageSection struct {
	Source      string `yaml:"source"`
	AspectRatio string `yaml:"aspectRatio"`
}

type VideoSection struct {
	Source   string `yaml:"source"`
	Autoplay bool   `yaml:"autoplay"`
}

type MultiWebSection struct {
	Sites []SiteEntry `yaml:"sites"`
}

type SiteEntry struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	Icon string `yaml:"icon"`
}

type PermissionsSection struct {
	Add    []string `yaml:"add"`
	Remove []string `yaml:"remove"`
}

type LocalesSection struct {
	Keep         []string `yaml:"keep"`
	StripLocales []string `yaml:"stripLocales"`
	StripScripts []string `yaml:"stripScripts"`
}

type FeaturesSection struct {
	Stacks    []string `yaml:"stacks"`
	GoRuntime bool     `yaml:"goRuntime"`
}

type TemplateSection struct {
	APK               string `yaml:"apk"`
	StripShellLocales bool   `yaml:"stripShellLocales"`
}

type SignSection struct {
	Keystore string `yaml:"keystore"`
	KsPass   string `yaml:"ksPass,omitempty"`
	KeyAlias string `yaml:"keyAlias,omitempty"`
	KeyPass  string `yaml:"keyPass,omitempty"`
	V1       bool   `yaml:"v1"`
	V2       bool   `yaml:"v2"`
	V3       bool   `yaml:"v3"`
	V4       bool   `yaml:"v4"`
	MinSDK   int    `yaml:"minSdk,omitempty"`
}

type OutputSection struct {
	Path  string `yaml:"path"`
	Align int    `yaml:"align"`
	Cache bool   `yaml:"cache"`
}
