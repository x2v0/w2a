package modules

// Enum values that must mirror the Kotlin source.
// Keep in sync with core/extension/ExtensionModule.kt.

var AllowedCategories = map[string]struct{}{
	"CONTENT_FILTER": {}, "CONTENT_ENHANCE": {}, "STYLE_MODIFIER": {},
	"THEME": {}, "FUNCTION_ENHANCE": {}, "AUTOMATION": {}, "NAVIGATION": {},
	"DATA_EXTRACT": {}, "DATA_SAVE": {}, "INTERACTION": {}, "ACCESSIBILITY": {},
	"MEDIA": {}, "VIDEO": {}, "IMAGE": {}, "AUDIO": {}, "SECURITY": {},
	"ANTI_TRACKING": {}, "SOCIAL": {}, "SHOPPING": {}, "READING": {},
	"TRANSLATE": {}, "DEVELOPER": {}, "OTHER": {},
}

var AllowedRunAt = map[string]struct{}{
	"DOCUMENT_START": {}, "DOCUMENT_END": {}, "DOCUMENT_IDLE": {},
	"CONTEXT_MENU": {}, "BEFORE_UNLOAD": {},
}

var AllowedPermissions = map[string]struct{}{
	"DOM_ACCESS": {}, "DOM_OBSERVE": {}, "CSS_INJECT": {}, "STORAGE": {},
	"COOKIE": {}, "INDEXED_DB": {}, "CACHE": {}, "NETWORK": {},
	"WEBSOCKET": {}, "FETCH_INTERCEPT": {}, "CLIPBOARD": {}, "NOTIFICATION": {},
	"ALERT": {}, "KEYBOARD": {}, "MOUSE": {}, "TOUCH": {}, "LOCATION": {},
	"CAMERA": {}, "MICROPHONE": {}, "DEVICE_INFO": {}, "MEDIA": {},
	"FULLSCREEN": {}, "PICTURE_IN_PICTURE": {}, "SCREEN_CAPTURE": {},
	"DOWNLOAD": {}, "FILE_ACCESS": {}, "EVAL": {}, "IFRAME": {},
	"WINDOW_OPEN": {}, "HISTORY": {}, "NAVIGATION": {},
}

var AllowedSourceTypes = map[string]struct{}{
	"CUSTOM": {}, "CHROME_EXTENSION": {},
}

// Plugin-json permission allowlist (lowercase in Python source).
var PluginJSONPermissions = map[string]struct{}{
	"STORAGE": {}, "FETCH": {}, "NOTIFY": {}, "BADGE": {},
	"CLIPBOARD": {}, "DOWNLOAD": {},
}

var PluginJSONRunAt = map[string]struct{}{
	"document_start": {}, "document_end": {}, "document_idle": {},
}
