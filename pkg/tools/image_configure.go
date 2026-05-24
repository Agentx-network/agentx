package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/providers"
)

// ConfigureImageProviderTool persists an image-generation provider's API key to
// config (tools.image.providers.<provider>), so the user can set up image
// generation entirely from chat: pick a provider, paste a key, generate. The
// key is written to the same config file the desktop Config → Images page uses,
// and image_generate (which reads providers live) picks it up on the next call.
type ConfigureImageProviderTool struct {
	configPath string
}

// NewConfigureImageProviderTool builds the tool. configPath is the config file
// to write (config.DefaultConfigPath() in normal operation).
func NewConfigureImageProviderTool(configPath string) *ConfigureImageProviderTool {
	return &ConfigureImageProviderTool{configPath: configPath}
}

func (t *ConfigureImageProviderTool) Name() string { return "configure_image_provider" }

func (t *ConfigureImageProviderTool) Description() string {
	return "Save an image-generation provider's API key. ONLY call this when the user has ALREADY " +
		"given you an actual API key in their message. NEVER call it to ask for a key, with an empty key, " +
		"or before trying image_generate — image_generate already uses the user's configured provider. " +
		"After saving a key here, call image_generate."
}

func (t *ConfigureImageProviderTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"provider": map[string]any{
				"type":        "string",
				"description": "Image provider to configure: 'gemini', 'openai', 'replicate', or 'seedance'.",
			},
			"api_key": map[string]any{
				"type":        "string",
				"description": "The API key / token the user provided for that provider.",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "Optional: a specific image model id to use (e.g. a Replicate 'owner/name' ref). Leave empty to use the provider default.",
			},
			"api_base": map[string]any{
				"type":        "string",
				"description": "Optional: a custom API base URL (for self-hosted or regional endpoints like Seedance/Ark).",
			},
		},
		"required": []string{"provider", "api_key"},
	}
}

func (t *ConfigureImageProviderTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	provider := strings.ToLower(strings.TrimSpace(asString(args["provider"])))
	apiKey := strings.TrimSpace(asString(args["api_key"]))
	model := strings.TrimSpace(asString(args["model"]))
	apiBase := strings.TrimSpace(asString(args["api_base"]))

	if provider == "" {
		return ErrorResult("provider is required (gemini, openai, replicate, or seedance)")
	}
	if apiKey == "" {
		return BlockedResult(
			"I need the API key to set up "+providerLabel(provider)+" for images. Please paste it.",
			"No api_key provided. Ask the user to paste the provider's API key, then call this tool again.",
		)
	}
	if !providers.ProviderSupportsImages(provider) {
		names := providers.ImageProviderNames()
		labels := make([]string, len(names))
		for i, n := range names {
			labels[i] = providerLabel(n)
		}
		return BlockedResult(
			fmt.Sprintf("%s can't generate images. Supported image providers are %s.", providerLabel(provider), strings.Join(labels, ", ")),
			fmt.Sprintf("Provider %q has no image capability. Ask the user to pick one of: %s.", provider, strings.Join(names, ", ")),
		)
	}

	cfg, err := config.LoadConfig(t.configPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("could not read config to save the key: %v", err))
	}
	if cfg.Tools.Image.Providers == nil {
		cfg.Tools.Image.Providers = map[string]config.ImageProviderConfig{}
	}

	// Already configured with this exact key → no-op. Stops the redundant
	// "ask for a key again" loop where the model re-runs configure for a
	// provider that's already set up instead of just generating.
	if existing, ok := cfg.Tools.Image.Providers[provider]; ok && existing.APIKey == apiKey {
		return &ToolResult{
			ForUser: fmt.Sprintf("%s is already set up for images. What would you like me to create?", providerLabel(provider)),
			ForLLM: fmt.Sprintf("%s is already configured with this key — do NOT ask for a key again. "+
				"If the user already gave an image subject, call image_generate now; otherwise ask what to depict.", provider),
			IsError: false,
		}
	}
	cfg.Tools.Image.Providers[provider] = config.ImageProviderConfig{
		APIKey:  apiKey,
		Model:   model,
		APIBase: apiBase,
	}
	if err := config.SaveConfig(t.configPath, cfg); err != nil {
		return ErrorResult(fmt.Sprintf("could not save the key to config: %v", err))
	}

	model = strings.TrimSpace(model)
	if model == "" {
		model = providers.DefaultImageModel(provider)
	}
	return &ToolResult{
		ForUser: fmt.Sprintf("Saved your %s key for image generation (model: %s). What would you like me to create?",
			providerLabel(provider), model),
		ForLLM: fmt.Sprintf("Saved %s image key (model %s). If the user already gave an image prompt, call image_generate now; "+
			"otherwise ask what image they want.", provider, model),
		IsError: false,
	}
}
