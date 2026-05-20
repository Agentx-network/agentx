package providers

import "testing"

func TestProviderFromModelRef(t *testing.T) {
	cases := map[string]string{
		"gemini/gemini-2.5-flash": "gemini",
		"openai/gpt-4o":           "openai",
		"OpenAI/GPT-4O":           "openai",
		"cerebras":                "cerebras",
		"gemini/gemini-2.5-image": "gemini",
		"replicate/black/forest":  "replicate",
	}
	for in, want := range cases {
		if got := ProviderFromModelRef(in); got != want {
			t.Errorf("ProviderFromModelRef(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProviderSupportsImages(t *testing.T) {
	supported := []string{"gemini", "google", "openai", "gemini/gemini-2.5-flash", "openai/gpt-4o"}
	for _, p := range supported {
		if !ProviderSupportsImages(p) {
			t.Errorf("ProviderSupportsImages(%q) = false, want true", p)
		}
	}
	unsupported := []string{"anthropic", "groq", "cerebras", "deepseek", "mistral", "ollama", ""}
	for _, p := range unsupported {
		if ProviderSupportsImages(p) {
			t.Errorf("ProviderSupportsImages(%q) = true, want false", p)
		}
	}
}

func TestDefaultImageModel(t *testing.T) {
	if got := DefaultImageModel("gemini"); got != "gemini-2.5-flash-image" {
		t.Errorf("DefaultImageModel(gemini) = %q, want gemini-2.5-flash-image", got)
	}
	if got := DefaultImageModel("openai"); got != "gpt-image-1" {
		t.Errorf("DefaultImageModel(openai) = %q, want gpt-image-1", got)
	}
	if got := DefaultImageModel("anthropic"); got != "" {
		t.Errorf("DefaultImageModel(anthropic) = %q, want empty", got)
	}
}

func TestIsImageModel(t *testing.T) {
	type pm struct{ provider, model string }
	valid := []pm{
		{"gemini", "gemini-2.5-flash-image"},
		{"openai", "gpt-image-1"},
		{"openai/gpt-4o", "dall-e-3"},
	}
	for _, c := range valid {
		if !IsImageModel(c.provider, c.model) {
			t.Errorf("IsImageModel(%q, %q) = false, want true", c.provider, c.model)
		}
	}
	// Hallucinated / wrong-provider / empty model names must be rejected.
	bad := []pm{
		{"gemini", "stable diffusion"},
		{"gemini", "gpt-image-1"},
		{"openai", ""},
		{"anthropic", "claude"},
	}
	for _, c := range bad {
		if IsImageModel(c.provider, c.model) {
			t.Errorf("IsImageModel(%q, %q) = true, want false", c.provider, c.model)
		}
	}
}

func TestImageModelsForReturnsCopy(t *testing.T) {
	got := ImageModelsFor("gemini")
	if len(got) == 0 {
		t.Fatal("expected models for gemini")
	}
	got[0] = "mutated"
	if ImageModelsFor("gemini")[0] == "mutated" {
		t.Error("ImageModelsFor returned a slice aliasing the package map; want a copy")
	}
}
