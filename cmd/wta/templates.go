package main

import "fmt"

func starterYAML(typ, name, pkg, url string) string {
	// Map alias types to canonical types (scaffold writes html-project/)
	yamlTyp := typ
	switch typ {
	case "chat", "notes", "dashboard", "rss", "reader":
		yamlTyp = "html"
	}

	header := fmt.Sprintf(`app:
  name: %q
  package: %s
  versionCode: 1
  versionName: "1.0.0"
  icon: icon.png
  engine: webview

type: %s

`, name, pkg, yamlTyp)

	switch typ {
	case "web":
		return header + fmt.Sprintf(`web:
  url: %s

output:
  path: out.apk
  align: 4
  cache: true
`, url)
	case "html", "chat", "notes", "dashboard", "rss", "reader":
		return header + `html:
  source: ./html-project
  entryFile: index.html

output:
  path: out.apk
  align: 4
`
	case "frontend":
		return header + `frontend:
  source: ./dist

output:
  path: out.apk
  align: 4
`
	case "go":
		return header + `go:
  source: ./goproject
  abis: [arm64-v8a, x86_64]
  noCgo: true

output:
  path: out.apk
  align: 4
`
	case "gallery":
		return header + `gallery:
  source: ./photos
  columns: 3

output:
  path: out.apk
`
	case "image":
		return header + `image:
  source: photo.jpg
  aspectRatio: fit

output:
  path: out.apk
`
	case "video":
		return header + `video:
  source: video.mp4
  autoplay: false

output:
  path: out.apk
`
	case "multiweb":
		return header + `multiweb:
  sites:
    - { name: "Site A", url: "https://a.com" }
    - { name: "Site B", url: "https://b.com" }

output:
  path: out.apk
`
	default:
		return header + `# TODO: add type-specific section for type="` + typ + `"

output:
  path: out.apk
`
	}
}
