package providers

import "strings"

// imageModelsByProvider maps a provider prefix (the part before "/" in a model
// ref like "gemini/gemini-2.5-flash") to the image-generation models it offers.
// First entry per provider is the default. Providers absent from this map have
// no image-output capability (Anthropic, Groq, Cerebras, DeepSeek, Mistral).
var imageModelsByProvider = map[string][]string{
	"gemini": {"gemini-3-pro-image-preview", "gemini-3.1-flash-image-preview", "gemini-2.5-flash-image"},
	"google": {"gemini-3-pro-image-preview", "gemini-3.1-flash-image-preview", "gemini-2.5-flash-image"},
	"openai": {"gpt-image-1", "dall-e-3"},
	// Seedance via Volcengine; Replicate/Stability via their APIs. Added as
	// their adapters land (see pkg/image).
	"seedance":  {"seedance-1.0"},
	"replicate": {"black-forest-labs/flux-1.1-pro", "stability-ai/sdxl"},
}

// ProviderFromModelRef extracts the provider prefix from a model reference.
// "gemini/gemini-2.5-flash" → "gemini"; "openai" → "openai".
func ProviderFromModelRef(modelRef string) string {
	if i := strings.Index(modelRef, "/"); i >= 0 {
		return strings.ToLower(modelRef[:i])
	}
	return strings.ToLower(modelRef)
}

// ImageModelsFor returns the image-generation models available for a provider
// (or a full model ref — the prefix is extracted). Empty slice if the provider
// has no image capability.
func ImageModelsFor(providerOrModelRef string) []string {
	p := ProviderFromModelRef(providerOrModelRef)
	models := imageModelsByProvider[p]
	out := make([]string, len(models))
	copy(out, models)
	return out
}

// ProviderSupportsImages reports whether a provider can generate images.
func ProviderSupportsImages(providerOrModelRef string) bool {
	return len(ImageModelsFor(providerOrModelRef)) > 0
}

// DefaultImageModel returns the preferred image model for a provider, or "".
func DefaultImageModel(providerOrModelRef string) string {
	if m := ImageModelsFor(providerOrModelRef); len(m) > 0 {
		return m[0]
	}
	return ""
}

// IsImageModel reports whether model is a known image model for the provider.
// Used to reject hallucinated model names (e.g. a weak LLM passing "stable
// diffusion") before they reach the provider API.
func IsImageModel(providerOrModelRef, model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	for _, m := range ImageModelsFor(providerOrModelRef) {
		if strings.EqualFold(m, model) {
			return true
		}
	}
	return false
}
