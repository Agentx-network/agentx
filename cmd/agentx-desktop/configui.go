package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Agentx-network/agentx/pkg/config"
)

//go:embed catalog.json
var catalogJSON []byte

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
	var providers []ProviderOption
	if err := json.Unmarshal(catalogJSON, &providers); err != nil {
		// Fallback: return empty list on parse error
		return nil
	}
	return providers
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

// QuickSetupChannel enables a channel with its token in one call.
func (c *ConfigService) QuickSetupChannel(channel string, token string) error {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	switch channel {
	case "telegram":
		cfg.Channels.Telegram.Enabled = true
		cfg.Channels.Telegram.Token = token
	case "discord":
		cfg.Channels.Discord.Enabled = true
		cfg.Channels.Discord.Token = token
	case "slack":
		cfg.Channels.Slack.Enabled = true
		cfg.Channels.Slack.BotToken = token
	default:
		return fmt.Errorf("unsupported channel for quick setup: %s", channel)
	}
	return config.SaveConfig(getConfigPath(), cfg)
}
