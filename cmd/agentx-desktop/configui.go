package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Agentx-network/agentx/pkg/config"
)

type ProviderOption struct {
	Name      string `json:"name"`
	ID        string `json:"id"`
	ModelName string `json:"modelName"`
	Model     string `json:"model"`
	APIBase   string `json:"apiBase"`
	KeyURL    string `json:"keyURL"`
	NeedsKey  bool   `json:"needsKey"`
}

type ConfigService struct {
	ctx context.Context
}

func NewConfigService() *ConfigService {
	return &ConfigService{}
}

func (c *ConfigService) startup(ctx context.Context) {
	c.ctx = ctx
}

func (c *ConfigService) GetConfig() (*config.Config, error) {
	return config.LoadConfig(getConfigPath())
}

func (c *ConfigService) SaveConfig(cfg *config.Config) error {
	path := getConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return config.SaveConfig(path, cfg)
}

func (c *ConfigService) GetModelList() ([]config.ModelConfig, error) {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return nil, err
	}
	return cfg.ModelList, nil
}

func (c *ConfigService) AddModel(model config.ModelConfig) error {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	cfg.ModelList = append(cfg.ModelList, model)
	return config.SaveConfig(getConfigPath(), cfg)
}

func (c *ConfigService) UpdateModel(index int, model config.ModelConfig) error {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	if index < 0 || index >= len(cfg.ModelList) {
		return fmt.Errorf("model index %d out of range", index)
	}
	cfg.ModelList[index] = model
	return config.SaveConfig(getConfigPath(), cfg)
}

func (c *ConfigService) RemoveModel(index int) error {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	if index < 0 || index >= len(cfg.ModelList) {
		return fmt.Errorf("model index %d out of range", index)
	}
	cfg.ModelList = append(cfg.ModelList[:index], cfg.ModelList[index+1:]...)
	return config.SaveConfig(getConfigPath(), cfg)
}

func (c *ConfigService) SetChannelEnabled(channel string, enabled bool) error {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	switch channel {
	case "telegram":
		cfg.Channels.Telegram.Enabled = enabled
	case "discord":
		cfg.Channels.Discord.Enabled = enabled
	case "slack":
		cfg.Channels.Slack.Enabled = enabled
	case "whatsapp":
		cfg.Channels.WhatsApp.Enabled = enabled
	case "feishu":
		cfg.Channels.Feishu.Enabled = enabled
	case "dingtalk":
		cfg.Channels.DingTalk.Enabled = enabled
	case "qq":
		cfg.Channels.QQ.Enabled = enabled
	case "line":
		cfg.Channels.LINE.Enabled = enabled
	case "onebot":
		cfg.Channels.OneBot.Enabled = enabled
	case "wecom":
		cfg.Channels.WeCom.Enabled = enabled
	case "wecom_app":
		cfg.Channels.WeComApp.Enabled = enabled
	case "maixcam":
		cfg.Channels.MaixCam.Enabled = enabled
	default:
		return fmt.Errorf("unknown channel: %s", channel)
	}
	return config.SaveConfig(getConfigPath(), cfg)
}

func (c *ConfigService) GetAgentDefaults() (*config.AgentDefaults, error) {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return nil, err
	}
	return &cfg.Agents.Defaults, nil
}

func (c *ConfigService) UpdateAgentDefaults(defaults config.AgentDefaults) error {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	cfg.Agents.Defaults = defaults
	return config.SaveConfig(getConfigPath(), cfg)
}

func (c *ConfigService) GetAvailableProviders() []ProviderOption {
	return []ProviderOption{
		{Name: "OpenAI", ID: "openai", ModelName: "gpt-5.2", Model: "openai/gpt-5.2", APIBase: "https://api.openai.com/v1", KeyURL: "https://platform.openai.com/api-keys", NeedsKey: true},
		{Name: "Anthropic", ID: "anthropic", ModelName: "claude-sonnet-4.6", Model: "anthropic/claude-sonnet-4.6", APIBase: "https://api.anthropic.com/v1", KeyURL: "https://console.anthropic.com/settings/keys", NeedsKey: true},
		{Name: "Google Gemini", ID: "gemini", ModelName: "gemini-2.0-flash", Model: "gemini/gemini-2.0-flash-exp", APIBase: "https://generativelanguage.googleapis.com/v1beta", KeyURL: "https://ai.google.dev/", NeedsKey: true},
		{Name: "DeepSeek", ID: "deepseek", ModelName: "deepseek-chat", Model: "deepseek/deepseek-chat", APIBase: "https://api.deepseek.com/v1", KeyURL: "https://platform.deepseek.com/", NeedsKey: true},
		{Name: "Groq", ID: "groq", ModelName: "llama-3.3-70b", Model: "groq/llama-3.3-70b-versatile", APIBase: "https://api.groq.com/openai/v1", KeyURL: "https://console.groq.com/keys", NeedsKey: true},
		{Name: "OpenRouter", ID: "openrouter", ModelName: "openrouter-auto", Model: "openrouter/auto", APIBase: "https://openrouter.ai/api/v1", KeyURL: "https://openrouter.ai/keys", NeedsKey: true},
		{Name: "Ollama (local)", ID: "ollama", ModelName: "llama3", Model: "ollama/llama3", APIBase: "http://localhost:11434/v1", NeedsKey: false},
		{Name: "Mistral AI", ID: "mistral", ModelName: "mistral-small", Model: "mistral/mistral-small-latest", APIBase: "https://api.mistral.ai/v1", KeyURL: "https://console.mistral.ai/api-keys", NeedsKey: true},
		{Name: "Cerebras", ID: "cerebras", ModelName: "cerebras-llama-3.3-70b", Model: "cerebras/llama-3.3-70b", APIBase: "https://api.cerebras.ai/v1", KeyURL: "https://inference.cerebras.ai/", NeedsKey: true},
	}
}

func (c *ConfigService) QuickSetupProvider(providerID string, apiKey string) error {
	providers := c.GetAvailableProviders()
	var provider *ProviderOption
	for i := range providers {
		if providers[i].ID == providerID {
			provider = &providers[i]
			break
		}
	}
	if provider == nil {
		return fmt.Errorf("unknown provider: %s", providerID)
	}

	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}

	key := apiKey
	if key == "" && !provider.NeedsKey {
		key = "ollama"
	}

	newModel := config.ModelConfig{
		ModelName: provider.ModelName,
		Model:     provider.Model,
		APIBase:   provider.APIBase,
		APIKey:    key,
	}

	// Check if this model already exists, update if so
	for i, m := range cfg.ModelList {
		if m.ModelName == provider.ModelName {
			cfg.ModelList[i] = newModel
			cfg.Agents.Defaults.ModelName = provider.ModelName
			return config.SaveConfig(getConfigPath(), cfg)
		}
	}

	cfg.ModelList = append(cfg.ModelList, newModel)
	cfg.Agents.Defaults.ModelName = provider.ModelName
	return config.SaveConfig(getConfigPath(), cfg)
}
