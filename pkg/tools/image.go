package tools

import (
	"context"
	"fmt"
	"os"
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
	resolve   func() []ImageProvider // currently-usable image providers (live from config)
	persist   func(ImageProvider)    // optional: save an auto-detected provider to the Images config
	outputDir string
}

// NewImageGenerateTool builds the tool. resolve returns the currently-usable
// image-capable providers (re-read from config each call); persist (optional)
// saves a provider that was auto-detected from the chat config into the
// dedicated Images config so it shows on the Config page; outputDir is where
// images are saved (typically <workspace>/images).
func NewImageGenerateTool(resolve func() []ImageProvider, persist func(ImageProvider), outputDir string) *ImageGenerateTool {
	return &ImageGenerateTool{resolve: resolve, persist: persist, outputDir: outputDir}
}

func (t *ImageGenerateTool) Name() string { return "image_generate" }

func (t *ImageGenerateTool) Description() string {
	return "Generate an image from a text prompt. ALWAYS call this FIRST whenever the user asks to " +
		"create/draw/generate/make a picture or image — it automatically uses the user's already-configured " +
		"AI provider (e.g. Gemini, OpenAI) and its key. Do NOT ask the user for an API key before calling this; " +
		"only if this tool replies that the current provider can't make images should you then ask for one. " +
		"Pass the subject the USER described in `prompt`; if the user did not say what to depict, ask them first " +
		"and never invent a subject. Returns the saved image path."
}

func (t *ImageGenerateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "The subject to depict, taken from what the USER asked for. Do NOT invent or assume a subject — if the user was vague, ask them first.",
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

	// Deterministic "ask first" guard. The prompt rule tells the model to ask
	// what to depict when the user didn't specify — but that's advisory, and
	// models (especially weaker ones) sometimes invent a subject anyway. This
	// catches the case in CODE: if the subject is vague/placeholder, refuse and
	// bounce the model back to ask the user, regardless of which model is in use.
	if isVaguePrompt(prompt) {
		return BlockedResult(
			"What would you like the image to show? Tell me the subject and I'll create it.",
			"The image subject is missing or too vague (the user didn't say what to depict). "+
				"Do NOT invent a subject. Ask the user what they want the image to show, then call image_generate "+
				"with their actual description.",
		)
	}

	// Resolve providers live so a key the user just added is visible.
	configured := t.resolve()

	// No image provider configured → guide the user through the self-service
	// flow: pick a provider, paste its key (configure_image_provider saves it),
	// then retry.
	if len(configured) == 0 {
		return BlockedResult(
			"Your current AI provider can't generate images. Which image provider should I use — "+
				"Gemini, OpenAI, or Replicate? Tell me which one and paste its API key, and I'll set it up and create your image.",
			"No image-capable provider is available (the user's current chat provider has no image model, and none is "+
				"configured). Tell the user their current provider can't make images and ask which image provider "+
				"(gemini/openai/replicate) to use + its API key. When they give a key, call configure_image_provider, "+
				"then image_generate again. Do NOT claim you generated an image until image_generate succeeds.",
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

	// Persist the provider we're about to use into the dedicated Images config,
	// so a provider auto-detected from the chat config (e.g. the user's Gemini
	// key) becomes visible/manageable on the Config → Images page. Idempotent;
	// best-effort (never blocks generation).
	if t.persist != nil {
		t.persist(ImageProvider{Provider: chosen.Provider, Model: model, APIKey: chosen.APIKey, APIBase: chosen.APIBase})
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

	// Verify the file actually landed on disk before telling the model it
	// succeeded — otherwise a silent write failure would have the model claim
	// "here's your image" with a path that renders as a broken preview.
	if info, statErr := os.Stat(res.Path); statErr != nil || info.Size() == 0 {
		return BlockedResult(
			"I generated the image but couldn't save it to disk. Please try again.",
			fmt.Sprintf("image.Generate returned path %q but the file is missing or empty (%v). "+
				"Do NOT claim the image was created; tell the user saving failed.", res.Path, statErr),
		)
	}

	// The IMAGE: marker lets the desktop chat render the file inline (Phase 3).
	msg := fmt.Sprintf("Generated your image with %s and saved it to:\nIMAGE:%s", chosen.Provider, res.Path)
	return UserResult(msg)
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

// isVaguePrompt reports whether an image subject is effectively unspecified —
// either a bare verb/placeholder ("something", "a picture", "an image") or too
// short to describe anything. Used to enforce "ask the user first" in code
// rather than trusting the model to obey the prompt rule.
func isVaguePrompt(prompt string) bool {
	p := strings.ToLower(strings.TrimSpace(prompt))
	p = strings.Trim(p, ".!?,'\"")
	switch p {
	case "", "something", "anything", "whatever", "a picture", "an image", "a image",
		"picture", "image", "a drawing", "drawing", "art", "some art", "generate",
		"create", "make one", "surprise me", "you decide", "your choice", "idk",
		"i don't know", "i dont know", "anything you want", "whatever you want":
		return true
	}
	// A "subject" with no real content (e.g. just "a", "the image of") can't
	// describe anything depictable.
	if len(p) < 3 {
		return true
	}
	return false
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
