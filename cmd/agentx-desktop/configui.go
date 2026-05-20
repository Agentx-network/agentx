package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/logger"
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
	if err := config.SaveConfig(path, cfg); err != nil {
		return err
	}
	notifyGatewayReload(cfg)
	return nil
}

// saveAndNotify persists config and POSTs /api/reload so the running gateway
// picks up changes without a manual restart.
func saveAndNotify(cfg *config.Config) error {
	if err := config.SaveConfig(getConfigPath(), cfg); err != nil {
		return err
	}
	notifyGatewayReload(cfg)
	return nil
}

// notifyGatewayReload POSTs /api/reload to the gateway. Failures are logged
// but not propagated — the config is already on disk and the next gateway
// start will pick it up.
func notifyGatewayReload(cfg *config.Config) {
	host := cfg.Gateway.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Gateway.Port
	if port == 0 {
		port = 18790
	}
	url := fmt.Sprintf("http://%s:%d/api/reload", host, port)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		logger.WarnCF("desktop", "Could not build gateway reload request",
			map[string]any{"error": err.Error()})
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		// Gateway may not be running yet (first launch before auto-start
		// completes). Don't surface as a failure to the user.
		logger.DebugCF("desktop", "Gateway reload skipped (gateway unreachable)",
			map[string]any{"url": url, "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.WarnCF("desktop", "Gateway reload returned non-200",
			map[string]any{"url": url, "status": resp.StatusCode})
	}
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
	return saveAndNotify(cfg)
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
	return saveAndNotify(cfg)
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
	return saveAndNotify(cfg)
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
	return saveAndNotify(cfg)
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
	return saveAndNotify(cfg)
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
			return saveAndNotify(cfg)
		}
	}

	cfg.ModelList = append(cfg.ModelList, newModel)
	cfg.Agents.Defaults.ModelName = provider.ModelName
	return saveAndNotify(cfg)
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
	return saveAndNotify(cfg)
}
