package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Agentx-network/agentx/pkg/image"
	"github.com/Agentx-network/agentx/pkg/providers"
)

// ImageProvider is a configured, image-capable provider the agent may use.
type ImageProvider struct {
	Provider string // "gemini", "openai", …
	Model    string // image model id; empty → adapter default
	APIKey   string
	APIBase  string
}

// ImageGenerateTool generates an image from a text prompt using one of the
// user's configured image-capable providers, saving the result to the
// workspace. Providers are resolved live on each call (via resolve) so a key
// the user adds mid-chat — through configure_image_provider or the Config page
// — is picked up immediately, without restarting the gateway.
type ImageGenerateTool struct {
	resolve   func() []ImageProvider
	outputDir string
}

// NewImageGenerateTool builds the tool. resolve returns the currently-configured
// image-capable providers (re-read from config each call); outputDir is where
// images are saved (typically <workspace>/images).
func NewImageGenerateTool(resolve func() []ImageProvider, outputDir string) *ImageGenerateTool {
	return &ImageGenerateTool{resolve: resolve, outputDir: outputDir}
}

func (t *ImageGenerateTool) Name() string { return "image_generate" }

func (t *ImageGenerateTool) Description() string {
	return "Generate an image from a text prompt using a configured image-capable provider " +
		"(e.g. Gemini, OpenAI). Returns the saved image file path. Only call this when the user " +
		"asks to create/draw/generate an image."
}

func (t *ImageGenerateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "Text description of the image to generate.",
			},
			"provider": map[string]any{
				"type":        "string",
				"description": "Optional: which configured image provider to use (e.g. 'gemini', 'openai'). Defaults to the first available.",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "Optional: specific image model to use. Defaults to the provider's preferred model.",
			},
		},
		"required": []string{"prompt"},
	}
}

func (t *ImageGenerateTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	prompt, _ := args["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ErrorResult("prompt is required to generate an image")
	}

	// Resolve providers live so a key the user just added is visible.
	configured := t.resolve()

	// No image provider configured → guide the user through the self-service
	// flow: pick a provider, paste its key (configure_image_provider saves it),
	// then retry.
	if len(configured) == 0 {
		return BlockedResult(
			"I can generate images, but no image provider is set up yet. Which would you like to use — "+
				"Gemini, OpenAI, or Replicate? Tell me the provider and paste its API key, and I'll save it and create your image.",
			"No image-capable provider is configured. Ask the user which provider (gemini/openai/replicate) "+
				"and for its API key. When they give a key, call configure_image_provider to save it, then call "+
				"image_generate again. Do NOT claim you generated an image until image_generate succeeds.",
		)
	}

	// Pick the provider: explicit request, else first configured. Weak models
	// often pass a provider that isn't configured (e.g. "openai" when only
	// "gemini" has a key); when there's only one configured provider we just
	// use it rather than bouncing the user with a needless question.
	requested := strings.ToLower(strings.TrimSpace(asString(args["provider"])))
	chosen := configured[0]
	if requested != "" {
		found := false
		for _, p := range configured {
			if p.Provider == requested {
				chosen = p
				found = true
				break
			}
		}
		if !found && len(configured) > 1 {
			avail := make([]string, 0, len(configured))
			for _, p := range configured {
				avail = append(avail, p.Provider)
			}
			return BlockedResult(
				fmt.Sprintf("I don't have %q configured for images. Available: %s. Which would you like?",
					requested, strings.Join(avail, ", ")),
				fmt.Sprintf("Requested image provider %q is not configured. Available: %s. Ask the user to pick one.",
					requested, strings.Join(avail, ", ")),
			)
		}
		// not found but only one provider → chosen already holds it.
	}

	// Only honour an explicitly-requested model if it's a real image model for
	// the chosen provider; weak LLMs hallucinate names ("stable diffusion"),
	// which the provider API rejects. Otherwise use the provider's default.
	model := strings.TrimSpace(asString(args["model"]))
	if !providers.IsImageModel(chosen.Provider, model) {
		model = chosen.Model
	}

	res, err := image.Generate(ctx, image.GenerateOptions{
		Provider:  chosen.Provider,
		Model:     model,
		APIKey:    chosen.APIKey,
		APIBase:   chosen.APIBase,
		Prompt:    prompt,
		OutputDir: t.outputDir,
	})
	if err != nil {
		forUser, forLLM := friendlyImageError(chosen.Provider, err)
		return BlockedResult(forUser, forLLM)
	}

	// The IMAGE: marker lets the desktop chat render the file inline (Phase 3).
	msg := fmt.Sprintf("Generated your image with %s and saved it to:\nIMAGE:%s", chosen.Provider, res.Path)
	return UserResult(msg)
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

// providerLabel returns a human-friendly provider name for messages.
func providerLabel(provider string) string {
	switch strings.ToLower(provider) {
	case "gemini", "google":
		return "Gemini"
	case "openai":
		return "OpenAI"
	case "replicate":
		return "Replicate"
	case "seedance":
		return "Seedance"
	default:
		if provider == "" {
			return "the provider"
		}
		return strings.ToUpper(provider[:1]) + provider[1:]
	}
}

// friendlyImageError turns a raw provider error into a clean user-facing
// message (forUser) and a short instruction for the model (forLLM). Image
// generation is a paid feature on every provider, so quota/billing failures
// are the common case and get a clear, non-technical explanation instead of
// the provider's raw error dump (URLs, metric names, retry seconds).
func friendlyImageError(provider string, err error) (forUser, forLLM string) {
	name := providerLabel(provider)
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "limit: 0") || strings.Contains(msg, "free_tier") || strings.Contains(msg, "billing"):
		forUser = fmt.Sprintf("%s image generation isn't available on a free API key — it needs billing enabled. "+
			"Turn on billing for your %s key, or add a paid OpenAI or Replicate key in Config → Images, then try again.", name, name)
		forLLM = fmt.Sprintf("%s image generation requires a paid/billing-enabled key (free tier limit is 0). "+
			"Tell the user to enable billing or configure a paid image provider. Do NOT retry automatically.", name)
	case strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") || strings.Contains(msg, "quota"):
		forUser = fmt.Sprintf("%s is rate-limiting image requests right now. Please wait a minute and try again.", name)
		forLLM = fmt.Sprintf("%s rate-limited the image request. Tell the user to wait and retry shortly.", name)
	case strings.Contains(msg, "api key") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "401") || strings.Contains(msg, "invalid_api_key"):
		forUser = fmt.Sprintf("The %s API key for image generation looks invalid. Check it in Config → Images.", name)
		forLLM = fmt.Sprintf("%s rejected the API key for image generation. Ask the user to verify the key in Config → Images.", name)
	default:
		clean := oneLine(err.Error())
		forUser = fmt.Sprintf("Image generation with %s failed: %s", name, clean)
		forLLM = forUser
	}
	return forUser, forLLM
}

// oneLine collapses whitespace/newlines and truncates a raw error for display.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const max = 200
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
