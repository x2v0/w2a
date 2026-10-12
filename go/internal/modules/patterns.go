package modules

import "regexp"

var (
	KebabCaseRE = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	SemverRE    = regexp.MustCompile(`^\d+(?:\.\d+){0,3}(?:[-+][\w.-]+)?$`)
	StoreIDRE   = regexp.MustCompile(`^[a-p]{32}$`)

	// PR reference "#123" preceded by ( or whitespace and followed by ) or word boundary.
	PRRefRE = regexp.MustCompile(`(?:\(|\s)#(\d+)(?:\)|\b)`)

	// <script type="hcj/page">...</script> (case-insensitive, DOTALL).
	HcjPageBlockRE = regexp.MustCompile(`(?is)<script[^>]*type\s*=\s*["']hcj/page["'][^>]*>(.*?)</script>`)
)

// Files allowed alongside the runtime manifest.
var AllowedExtraFiles = map[string]struct{}{
	"README.md": {}, "CHANGELOG.md": {}, "LICENSE": {}, "LICENSE.md": {},
	".gitkeep": {},
}

// Icon files contributors may drop next to plugin.html.
var AllowedIconFiles = map[string]struct{}{
	"icon.png": {}, "icon.svg": {}, "icon.webp": {},
	"icon.jpg": {}, "icon.jpeg": {},
}

const MaxIconBytes = 256 * 1024
const MaxPageScriptBytes = 512 * 1024

// Expected files inside a module folder (plugins the runtime downloads).
var ExpectedFolderFiles = map[string]struct{}{
	"plugin.json": {}, "plugin.html": {}, "main.js": {},
	"style.css": {}, "panel.html": {},
}
