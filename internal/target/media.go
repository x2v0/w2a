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
// Shell reads galleryConfig.items (list of GalleryShellItem):
//
//	{id, assetPath, type, name, duration}
//
// Files served from assets/gallery/<name>.
type GalleryTarget struct{}

func (t *GalleryTarget) Name() string { return "gallery" }

func (t *GalleryTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Gallery.Source)
	if resolved, err := filepath.EvalSymlinks(src); err == nil {
		src = resolved
	}
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("gallery: source not found: %s", src)
	}
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}

	var itemNames []string
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
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := a.AddAssetBytes("assets/gallery/"+rel, data); err != nil {
			return err
		}
		itemNames = append(itemNames, rel)
		n++
		return nil
	})
	if err != nil {
		return err
	}

	// Build GalleryShellConfig.items
	galleryItems := make([]map[string]any, 0, len(itemNames))
	for _, name := range itemNames {
		galleryItems = append(galleryItems, map[string]any{
			"id":        name,
			"assetPath": "gallery/" + name,
			"type":      "IMAGE",
			"name":      name,
			"duration":  0,
		})
	}

	cols := cfg.Gallery.Columns
	if cols == 0 {
		cols = 3
	}

	cfgJSON := map[string]any{
		"appType":  "GALLERY",
		"language": "RUSSIAN",
		"galleryConfig": map[string]any{
			"items":            galleryItems,
			"playMode":         "SEQUENTIAL",
			"imageInterval":    3,
			"loop":             true,
			"autoPlay":         false,
			"backgroundColor":  "#000000",
			"showThumbnailBar": true,
			"gridColumns":      cols,
		},
	}
	data, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/app_config.json", data)
	fmt.Printf("gallery: %d images\n", n)
	return a.SaveAPK(outPath)
}

// ImageTarget shows a single image.
// Shell reads assets/splash_media.png (fixed path, see ShellMediaContent.kt).
// Config: mediaConfig controls playback.
type ImageTarget struct{}

func (t *ImageTarget) Name() string { return "image" }

func (t *ImageTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Image.Source)
	if resolved, err := filepath.EvalSymlinks(src); err == nil {
		src = resolved
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("image: %w", err)
	}
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}

	// Shell opens media_content.png via openFd() → must be STORED.
	if err := a.AddAssetStoredBytes("assets/media_content.png", data); err != nil {
		return err
	}

	fillScreen := cfg.Image.AspectRatio != "fit"

	cfgJSON := map[string]any{
		"appType":  "IMAGE",
		"language": "RUSSIAN",
		"mediaConfig": map[string]any{
			"enableAudio":     false,
			"loop":            false,
			"autoPlay":        false,
			"fillScreen":      fillScreen,
			"landscape":       false,
			"keepScreenOn":    true,
			"backgroundColor": "#000000",
		},
	}
	body, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/app_config.json", body)
	fmt.Printf("image: %s (%d bytes)\n", src, len(data))
	return a.SaveAPK(outPath)
}

// VideoTarget shows a single video.
// Shell reads assets/splash_media.mp4 (fixed path).
type VideoTarget struct{}

func (t *VideoTarget) Name() string { return "video" }

func (t *VideoTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	src := cfg.Resolve(cfg.Video.Source)
	if resolved, err := filepath.EvalSymlinks(src); err == nil {
		src = resolved
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("video: %w", err)
	}
	a, err := apk.LoadAPK(templatePath)
	if err != nil {
		return err
	}

	// Shell opens media_content.mp4 via openFd() for MediaPlayer.
	// MUST be STORED (uncompressed), otherwise FileNotFoundException.
	if err := a.AddAssetStoredBytes("assets/media_content.mp4", data); err != nil {
		return err
	}

	cfgJSON := map[string]any{
		"appType":  "VIDEO",
		"language": "RUSSIAN",
		"mediaConfig": map[string]any{
			"enableAudio":     true,
			"loop":            true,
			"autoPlay":        cfg.Video.Autoplay,
			"fillScreen":      true,
			"landscape":       false,
			"keepScreenOn":    true,
			"backgroundColor": "#000000",
		},
	}
	body, _ := json.MarshalIndent(cfgJSON, "", "  ")
	_ = a.AddAssetBytes("assets/app_config.json", body)
	fmt.Printf("video: %s (%d bytes)\n", src, len(data))
	return a.SaveAPK(outPath)
}
