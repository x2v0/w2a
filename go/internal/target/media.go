package target

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
	"github.com/x2v0/w2a/go/internal/config"
)

// GalleryTarget bundles a directory of images.
type GalleryTarget struct{}

func (t *GalleryTarget) Name() string { return "gallery" }

func (t *GalleryTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Gallery.Source)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("gallery: source not found: %s", src)
	}
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}
	n := 0
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".heic":
		default:
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := a.AddAssetBytes("assets/gallery/"+filepath.ToSlash(rel), data); err != nil {
			return err
		}
		n++
		return nil
	})
	if err != nil {
		return err
	}
	cols := cfg.Gallery.Columns
	if cols == 0 {
		cols = 3
	}
	cfgJSON := map[string]any{"appType": "GALLERY", "columns": cols}
	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/webtoapp/config.json", data)
	fmt.Printf("gallery: %d images\n", n)
	return a.SaveAPK(outPath)
}

// ImageTarget shows a single image.
type ImageTarget struct{}

func (t *ImageTarget) Name() string { return "image" }

func (t *ImageTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Image.Source)
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("image: %w", err)
	}
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(src))
	if err := a.AddAssetBytes("assets/image"+ext, data); err != nil {
		return err
	}
	cfgJSON := map[string]any{
		"appType":     "IMAGE",
		"imageAsset":  "image" + ext,
		"aspectRatio": cfg.Image.AspectRatio,
	}
	body, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/webtoapp/config.json", body)
	fmt.Printf("image: %s (%d bytes)\n", src, len(data))
	return a.SaveAPK(outPath)
}

// VideoTarget shows a single video.
type VideoTarget struct{}

func (t *VideoTarget) Name() string { return "video" }

func (t *VideoTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Video.Source)
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("video: %w", err)
	}
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(src))
	if err := a.AddAssetBytes("assets/video"+ext, data); err != nil {
		return err
	}
	cfgJSON := map[string]any{
		"appType":    "VIDEO",
		"videoAsset": "video" + ext,
		"autoplay":   cfg.Video.Autoplay,
	}
	body, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/webtoapp/config.json", body)
	fmt.Printf("video: %s (%d bytes)\n", src, len(data))
	return a.SaveAPK(outPath)
}
