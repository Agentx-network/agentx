package image

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupported(t *testing.T) {
	for _, p := range []string{"gemini", "google", "GEMINI", "openai", "seedance", "replicate"} {
		if !Supported(p) {
			t.Errorf("Supported(%q) = false, want true", p)
		}
	}
	for _, p := range []string{"", "anthropic", "groq", "cerebras"} {
		if Supported(p) {
			t.Errorf("Supported(%q) = true, want false", p)
		}
	}
}

func TestGenerateValidation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		opts GenerateOptions
		want string
	}{
		{"no provider", GenerateOptions{Prompt: "x", APIKey: "k"}, "provider is required"},
		{"unknown provider", GenerateOptions{Provider: "midjourney", Prompt: "x", APIKey: "k"}, "no image adapter"},
		{"no prompt", GenerateOptions{Provider: "gemini", APIKey: "k"}, "prompt is required"},
		{"no key", GenerateOptions{Provider: "gemini", Prompt: "x"}, "API key"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Generate(ctx, c.opts)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("Generate err = %v, want containing %q", err, c.want)
			}
		})
	}
}

func TestExtForMime(t *testing.T) {
	cases := map[string]string{
		"image/png":  ".png",
		"image/jpeg": ".jpg",
		"image/webp": ".webp",
		"":           ".png",
	}
	for mime, want := range cases {
		if got := extForMime(mime); got != want {
			t.Errorf("extForMime(%q) = %q, want %q", mime, got, want)
		}
	}
}

// TestGenerateSavesFile exercises the save path with a stub adapter so we don't
// hit the network.
func TestGenerateSavesFile(t *testing.T) {
	orig := adapters["stub"]
	adapters["stub"] = func(ctx context.Context, opts GenerateOptions) ([]byte, string, error) {
		return []byte("\x89PNG fake"), "image/png", nil
	}
	defer func() {
		if orig == nil {
			delete(adapters, "stub")
		} else {
			adapters["stub"] = orig
		}
	}()

	dir := t.TempDir()
	res, err := Generate(context.Background(), GenerateOptions{
		Provider:  "stub",
		APIKey:    "k",
		Prompt:    "a cat",
		OutputDir: dir,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if filepath.Dir(res.Path) != dir {
		t.Errorf("saved to %q, want under %q", res.Path, dir)
	}
	if filepath.Ext(res.Path) != ".png" {
		t.Errorf("ext = %q, want .png", filepath.Ext(res.Path))
	}
	if _, err := os.Stat(res.Path); err != nil {
		t.Errorf("file not written: %v", err)
	}
}
