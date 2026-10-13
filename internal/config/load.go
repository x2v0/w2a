package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var ValidTypes = map[string]bool{
	"web": true, "html": true, "frontend": true,
	"nodejs": true, "go": true, "python": true,
	"php": true, "wordpress": true,
	"gallery": true, "image": true, "video": true,
	"multiweb": true,
}

var ValidEngines = map[string]bool{"webview": true, "geckoview": true, "": true}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config: yaml: %w", err)
	}
	abs, _ := filepath.Abs(path)
	c.BaseDir = filepath.Dir(abs)
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) Resolve(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.BaseDir, p)
}

func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("config: app.name is required")
	}
	if c.App.Package == "" {
		return fmt.Errorf("config: app.package is required")
	}
	if c.Type == "" {
		return fmt.Errorf("config: type is required")
	}
	if !ValidTypes[c.Type] {
		return fmt.Errorf("config: unknown type %q", c.Type)
	}
	if !ValidEngines[c.App.Engine] {
		return fmt.Errorf("config: unknown engine %q", c.App.Engine)
	}
	if c.Output.Path == "" {
		return fmt.Errorf("config: output.path is required")
	}
	switch c.Type {
	case "web":
		if c.Web == nil || c.Web.URL == "" {
			return fmt.Errorf("config: web.url is required for type=web")
		}
	case "html":
		if c.HTML == nil || c.HTML.Source == "" {
			return fmt.Errorf("config: html.source is required")
		}
	case "frontend":
		if c.Frontend == nil || c.Frontend.Source == "" {
			return fmt.Errorf("config: frontend.source is required")
		}
	case "nodejs":
		if c.NodeJS == nil || c.NodeJS.Source == "" {
			return fmt.Errorf("config: nodejs.source is required")
		}
	case "go":
		if c.Go == nil || c.Go.Source == "" {
			return fmt.Errorf("config: go.source is required")
		}
	case "python":
		if c.Python == nil || c.Python.Source == "" {
			return fmt.Errorf("config: python.source is required")
		}
	case "php":
		if c.PHP == nil || c.PHP.Source == "" {
			return fmt.Errorf("config: php.source is required")
		}
	case "wordpress":
		if c.WordPress == nil || c.WordPress.Source == "" {
			return fmt.Errorf("config: wordpress.source is required")
		}
	case "gallery":
		if c.Gallery == nil || c.Gallery.Source == "" {
			return fmt.Errorf("config: gallery.source is required")
		}
	case "image":
		if c.Image == nil || c.Image.Source == "" {
			return fmt.Errorf("config: image.source is required")
		}
	case "video":
		if c.Video == nil || c.Video.Source == "" {
			return fmt.Errorf("config: video.source is required")
		}
	case "multiweb":
		if c.MultiWeb == nil || len(c.MultiWeb.Sites) == 0 {
			return fmt.Errorf("config: multiweb.sites is required")
		}
	}
	return nil
}
