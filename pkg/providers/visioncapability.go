package providers

import "strings"

// Vision / audio INPUT capability — whether a model can accept image or audio
// files as input (distinct from image *generation*, which imagecapability.go
// covers). Used to decide whether an uploaded image/audio can be attached to
// the model turn as a FilePart, or whether we should fall back to a warning /
// transcription. Kept deliberately conservative: unknown → not supported, so we
// warn rather than send bytes a provider will reject.

// visionExcludedSubstrings are model-name fragments that, within an otherwise
// vision-capable provider, denote a text-only variant.
var visionExcludedSubstrings = []string{
	"instruct", // e.g. text-only instruct variants
}

// ProviderSupportsVision reports whether the given provider + model can accept
// image input. provider is the lowercased provider name (e.g. "openai",
// "anthropic", "gemini"/"google"); model is the model id.
func ProviderSupportsVision(provider, model string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	m := strings.ToLower(strings.TrimSpace(model))

	switch provider {
	case "anthropic":
		// Claude 3 and later are all vision-capable. Claude 2.x is not.
		return !strings.Contains(m, "claude-2") && !strings.Contains(m, "claude-instant")
	case "gemini", "google":
		// Gemini 1.5+ and 2.x are multimodal. The legacy text-only "gemini-pro"
		// (no version) is the only common exception.
		if m == "gemini-pro" || strings.HasPrefix(m, "gemini-pro-") {
			return false
		}
		return strings.Contains(m, "gemini")
	case "openai":
		// gpt-4o / gpt-4.1 / gpt-4-turbo / gpt-5 / o-series all take images.
		// gpt-3.5 does not.
		if strings.Contains(m, "gpt-3.5") || strings.Contains(m, "gpt-35") {
			return false
		}
		if strings.HasPrefix(m, "gpt-4o") || strings.HasPrefix(m, "gpt-4.1") ||
			strings.Contains(m, "gpt-4-turbo") || strings.HasPrefix(m, "gpt-5") ||
			strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") ||
			strings.HasPrefix(m, "chatgpt") {
			return !containsAny(m, visionExcludedSubstrings)
		}
		return false
	case "openrouter", "openai-compat", "openaicompat":
		// "auto" is a router that selects a model per-request (and picks a
		// vision-capable one when the request carries images), so attach
		// optimistically rather than warn. Otherwise trust model-name hints.
		if m == "auto" || strings.Contains(m, "auto") {
			return true
		}
		return strings.Contains(m, "vision") || strings.Contains(m, "gpt-4o") ||
			strings.Contains(m, "claude-3") || strings.Contains(m, "gemini") ||
			strings.Contains(m, "llava") || strings.Contains(m, "pixtral") ||
			strings.Contains(m, "qwen-vl") || strings.Contains(m, "qwen2-vl")
	default:
		return false
	}
}

// ProviderSupportsAudioInput reports whether the provider + model can accept
// audio file input directly (not via transcription). Only OpenAI audio-preview
// models and Gemini currently do; Anthropic does not.
func ProviderSupportsAudioInput(provider, model string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	m := strings.ToLower(strings.TrimSpace(model))

	switch provider {
	case "gemini", "google":
		return strings.Contains(m, "gemini") &&
			!(m == "gemini-pro" || strings.HasPrefix(m, "gemini-pro-"))
	case "openai":
		// Only the audio-capable variants accept input_audio blocks.
		return strings.Contains(m, "audio") || strings.HasPrefix(m, "gpt-4o-audio")
	default:
		return false
	}
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
