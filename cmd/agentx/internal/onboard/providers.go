package onboard

import "github.com/Agentx-network/agentx/pkg/config"

type providerInfo struct {
	Name       string // Display name
	ID         string // Internal identifier
	ModelName  string // User-facing model alias
	Model      string // Protocol/model-identifier
	APIBase    string // API endpoint URL
	DefaultKey string // Pre-filled API key (e.g. "ollama")
	KeyURL     string // URL to obtain API key
	NeedsKey   bool   // Whether an API key is required
}

var providers = []providerInfo{
	{
		Name:     "OpenAI",
		ID:       "openai",
		ModelName: "gpt-5.2",
		Model:    "openai/gpt-5.2",
		APIBase:  "https://api.openai.com/v1",
		KeyURL:   "https://platform.openai.com/api-keys",
		NeedsKey: true,
	},
	{
		Name:     "Anthropic",
		ID:       "anthropic",
		ModelName: "claude-sonnet-4.6",
		Model:    "anthropic/claude-sonnet-4.6",
		APIBase:  "https://api.anthropic.com/v1",
		KeyURL:   "https://console.anthropic.com/settings/keys",
		NeedsKey: true,
	},
	{
		Name:     "Google Gemini",
		ID:       "gemini",
		ModelName: "gemini-2.0-flash",
		Model:    "gemini/gemini-2.0-flash-exp",
		APIBase:  "https://generativelanguage.googleapis.com/v1beta",
		KeyURL:   "https://ai.google.dev/",
		NeedsKey: true,
	},
	{
		Name:     "DeepSeek",
		ID:       "deepseek",
		ModelName: "deepseek-chat",
		Model:    "deepseek/deepseek-chat",
		APIBase:  "https://api.deepseek.com/v1",
		KeyURL:   "https://platform.deepseek.com/",
		NeedsKey: true,
	},
	{
		Name:     "Groq",
		ID:       "groq",
		ModelName: "llama-3.3-70b",
		Model:    "groq/llama-3.3-70b-versatile",
		APIBase:  "https://api.groq.com/openai/v1",
		KeyURL:   "https://console.groq.com/keys",
		NeedsKey: true,
	},
	{
		Name:     "OpenRouter (100+ models)",
		ID:       "openrouter",
		ModelName: "openrouter-auto",
		Model:    "openrouter/auto",
		APIBase:  "https://openrouter.ai/api/v1",
		KeyURL:   "https://openrouter.ai/keys",
		NeedsKey: true,
	},
	{
		Name:       "Ollama (local)",
		ID:         "ollama",
		ModelName:  "llama3",
		Model:      "ollama/llama3",
		APIBase:    "http://localhost:11434/v1",
		DefaultKey: "ollama",
		NeedsKey:   false,
	},
	{
		Name:     "Mistral AI",
		ID:       "mistral",
		ModelName: "mistral-small",
		Model:    "mistral/mistral-small-latest",
		APIBase:  "https://api.mistral.ai/v1",
		KeyURL:   "https://console.mistral.ai/api-keys",
		NeedsKey: true,
	},
	{
		Name:     "Cerebras",
		ID:       "cerebras",
		ModelName: "cerebras-llama-3.3-70b",
		Model:    "cerebras/llama-3.3-70b",
		APIBase:  "https://api.cerebras.ai/v1",
		KeyURL:   "https://inference.cerebras.ai/",
		NeedsKey: true,
	},
}

func findProvider(id string) *providerInfo {
	for i := range providers {
		if providers[i].ID == id {
			return &providers[i]
		}
	}
	return nil
}

func buildModelConfig(p *providerInfo, apiKey string) config.ModelConfig {
	key := apiKey
	if key == "" {
		key = p.DefaultKey
	}
	return config.ModelConfig{
		ModelName: p.ModelName,
		Model:     p.Model,
		APIBase:   p.APIBase,
		APIKey:    key,
	}
}
