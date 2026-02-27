package providers

import (
	"context"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/google"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openaicompat"

	"github.com/Agentx-network/agentx/pkg/config"
)

// FantasyModelFromConfig creates a Fantasy LanguageModel from a ModelConfig.
// It maps the protocol prefix to the appropriate Fantasy provider.
func FantasyModelFromConfig(cfg *config.ModelConfig) (fantasy.LanguageModel, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	// Trim whitespace from API key to prevent auth failures
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)

	protocol, modelID := ExtractProtocol(cfg.Model)

	provider, err := createFantasyProvider(protocol, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating fantasy provider for %s: %w", protocol, err)
	}

	model, err := provider.LanguageModel(context.Background(), modelID)
	if err != nil {
		return nil, fmt.Errorf("creating language model %s: %w", modelID, err)
	}

	return model, nil
}

// createFantasyProvider creates the appropriate Fantasy provider for a given protocol.
func createFantasyProvider(protocol string, cfg *config.ModelConfig) (fantasy.Provider, error) {
	switch protocol {
	case "anthropic":
		return createAnthropicProvider(cfg)

	case "openai":
		return createOpenAIProvider(cfg)

	case "gemini", "google":
		return createGoogleProvider(cfg)

	case "openrouter", "groq", "deepseek", "ollama", "mistral", "cerebras",
		"qwen", "vllm", "nvidia", "moonshot", "zhipu", "volcengine",
		"shengsuanyun":
		return createOpenAICompatProvider(protocol, cfg)

	default:
		return nil, fmt.Errorf("unsupported protocol %q", protocol)
	}
}

func createAnthropicProvider(cfg *config.ModelConfig) (fantasy.Provider, error) {
	var opts []anthropic.Option
	if cfg.APIKey != "" {
		opts = append(opts, anthropic.WithAPIKey(cfg.APIKey))
	}
	if cfg.APIBase != "" {
		opts = append(opts, anthropic.WithBaseURL(cfg.APIBase))
	}
	return anthropic.New(opts...)
}

func createOpenAIProvider(cfg *config.ModelConfig) (fantasy.Provider, error) {
	var opts []openai.Option
	if cfg.APIKey != "" {
		opts = append(opts, openai.WithAPIKey(cfg.APIKey))
	}
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = getDefaultAPIBase("openai")
	}
	if apiBase != "" {
		opts = append(opts, openai.WithBaseURL(apiBase))
	}
	return openai.New(opts...)
}

func createGoogleProvider(cfg *config.ModelConfig) (fantasy.Provider, error) {
	var opts []google.Option
	if cfg.APIKey != "" {
		opts = append(opts, google.WithGeminiAPIKey(cfg.APIKey))
	}
	if cfg.APIBase != "" {
		opts = append(opts, google.WithBaseURL(cfg.APIBase))
	}
	return google.New(opts...)
}

func createOpenAICompatProvider(protocol string, cfg *config.ModelConfig) (fantasy.Provider, error) {
	var opts []openaicompat.Option
	if cfg.APIKey != "" {
		opts = append(opts, openaicompat.WithAPIKey(cfg.APIKey))
	}
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = getDefaultAPIBase(protocol)
	}
	if apiBase != "" {
		opts = append(opts, openaicompat.WithBaseURL(apiBase))
	}
	opts = append(opts, openaicompat.WithName(protocol))
	return openaicompat.New(opts...)
}

// FantasyModelFromFullConfig creates a Fantasy LanguageModel from the full Config.
// This resolves the model from model_list and creates the appropriate provider.
func FantasyModelFromFullConfig(cfg *config.Config) (fantasy.LanguageModel, error) {
	model := cfg.Agents.Defaults.GetModelName()

	// Ensure model_list is populated
	if cfg.HasProvidersConfig() {
		providerModels := config.ConvertProvidersToModelList(cfg)
		existingModelNames := make(map[string]bool)
		for _, m := range cfg.ModelList {
			existingModelNames[m.ModelName] = true
		}
		for _, pm := range providerModels {
			if !existingModelNames[pm.ModelName] {
				cfg.ModelList = append(cfg.ModelList, pm)
			}
		}
	}

	if len(cfg.ModelList) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, fmt.Errorf("model %q not found in model_list: %w", model, err)
	}

	return FantasyModelFromConfig(modelCfg)
}
