package providers

import (
	"sort"
	"strings"
)

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

// ImageProviderNames returns the canonical list of provider names that can
// generate images, in a deterministic order. "google" is collapsed into
// "gemini" so the UI doesn't show both. Used by the image tool when it needs to
// tell the user which providers they can pick from.
func ImageProviderNames() []string {
	seen := map[string]bool{"google": true} // skip — duplicate of gemini
	out := []string{}
	for _, p := range []string{"gemini", "openai", "replicate", "seedance"} {
		if _, ok := imageModelsByProvider[p]; ok && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	// Append any others present in the map but not in the preferred order above
	// (forward-compatibility for new image providers).
	keys := make([]string, 0, len(imageModelsByProvider))
	for k := range imageModelsByProvider {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
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
