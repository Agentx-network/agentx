// Package image generates images via provider APIs and saves them to disk.
// Each provider has an adapter (gemini.go, openai.go, …) selected by the
// Provider field. The agent's image_generate tool is the main caller.
package image

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenerateOptions configures a single image-generation request.
type GenerateOptions struct {
	Provider  string // "gemini", "openai", "seedance", "replicate"
	Model     string // provider-specific model ID; empty → adapter default
	APIKey    string // provider API key (required for all current adapters)
	APIBase   string // optional base URL override (Seedance/self-hosted)
	Prompt    string // text description of the image
	OutputDir string // directory to save the PNG; created if missing
}

// Result describes a generated image.
type Result struct {
	Path     string // absolute path to the saved file
	MimeType string // e.g. "image/png"
	Provider string
	Model    string
}

// adapter is the per-provider generation function. It returns the raw image
// bytes and mime type; saving is handled centrally by Generate.
type adapter func(ctx context.Context, opts GenerateOptions) (data []byte, mime string, err error)

var adapters = map[string]adapter{
	"gemini":    generateGemini,
	"google":    generateGemini,
	"openai":    generateOpenAI,
	"seedance":  generateSeedance,
	"replicate": generateReplicate,
}

// Supported reports whether an adapter exists for the provider.
func Supported(provider string) bool {
	_, ok := adapters[strings.ToLower(provider)]
	return ok
}

// Generate produces an image and saves it under OutputDir. Returns the saved
// file's metadata. The prompt and API key are required; the model defaults to
// the provider's preferred image model when empty.
func Generate(ctx context.Context, opts GenerateOptions) (*Result, error) {
	provider := strings.ToLower(strings.TrimSpace(opts.Provider))
	if provider == "" {
		return nil, fmt.Errorf("image: provider is required")
	}
	gen, ok := adapters[provider]
	if !ok {
		return nil, fmt.Errorf("image: provider %q has no image adapter", provider)
	}
	if strings.TrimSpace(opts.Prompt) == "" {
		return nil, fmt.Errorf("image: prompt is required")
	}
	if strings.TrimSpace(opts.APIKey) == "" {
		return nil, fmt.Errorf("image: API key for %s is required", provider)
	}

	data, mime, err := gen(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("image: %s returned no image data", provider)
	}

	outDir := opts.OutputDir
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("image: create output dir: %w", err)
	}

	ext := extForMime(mime)
	name := fmt.Sprintf("img-%s%s", time.Now().Format("20060102-150405"), ext)
	path := filepath.Join(outDir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("image: save file: %w", err)
	}
	abs, _ := filepath.Abs(path)

	return &Result{Path: abs, MimeType: mime, Provider: provider, Model: opts.Model}, nil
}

func extForMime(mime string) string {
	switch {
	case strings.Contains(mime, "jpeg"), strings.Contains(mime, "jpg"):
		return ".jpg"
	case strings.Contains(mime, "webp"):
		return ".webp"
	default:
		return ".png"
	}
}
