package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Agentx-network/agentx/pkg/agent"
	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/logger"
	"github.com/Agentx-network/agentx/pkg/providers"
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

// ImageProviderInfo is the UI-facing view of one configured image provider.
type ImageProviderInfo struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
	APIBase  string `json:"api_base"`
}

// GetImageProviders returns the dedicated image providers configured under
// tools.image.providers, sorted by provider name for stable UI ordering.
func (c *ConfigService) GetImageProviders() ([]ImageProviderInfo, error) {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return nil, err
	}
	out := make([]ImageProviderInfo, 0, len(cfg.Tools.Image.Providers))
	for name, ip := range cfg.Tools.Image.Providers {
		out = append(out, ImageProviderInfo{
			Provider: name,
			Model:    ip.Model,
			APIKey:   ip.APIKey,
			APIBase:  ip.APIBase,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Provider < out[j].Provider })
	return out, nil
}

// SetImageProvider adds or updates the API key (and optional model/base) for a
// dedicated image provider. Provider name is lower-cased for consistency.
func (c *ConfigService) SetImageProvider(provider, apiKey, model, apiBase string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return fmt.Errorf("provider name is required")
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API key is required for %s", provider)
	}
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	if cfg.Tools.Image.Providers == nil {
		cfg.Tools.Image.Providers = map[string]config.ImageProviderConfig{}
	}
	cfg.Tools.Image.Providers[provider] = config.ImageProviderConfig{
		APIKey:  strings.TrimSpace(apiKey),
		Model:   strings.TrimSpace(model),
		APIBase: strings.TrimSpace(apiBase),
	}
	return saveAndNotify(cfg)
}

// RemoveImageProvider deletes a dedicated image provider's config.
func (c *ConfigService) RemoveImageProvider(provider string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	delete(cfg.Tools.Image.Providers, provider)
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

// ListProviderModels fetches the provider's live model list (Gemini, OpenAI,
// OpenRouter) so the UI can show current models — including ones newer than the
// shipped catalog — instead of a hardcoded list. provider is the AgentX prefix
// ("gemini"/"openai"/"openrouter"); apiBase may be empty to use the default.
// Returns an error for unsupported providers or on network/auth failure, so the
// UI can fall back to the static catalog.
func (c *ConfigService) ListProviderModels(provider, apiBase, apiKey string) ([]providers.DiscoveredModel, error) {
	return providers.ListModels(context.Background(), provider, apiBase, apiKey)
}

// ValidateProviderKey probes the provider with the entered key so a bad key is
// caught at onboarding instead of at first chat. Returns an empty string when
// the key checks out, or a short user-facing reason when it doesn't. Providers
// without a stable validation endpoint return "" (skipped, not failed).
func (c *ConfigService) ValidateProviderKey(provider, apiBase, apiKey string) string {
	if err := providers.ValidateKey(context.Background(), provider, apiBase, apiKey); err != nil {
		// Reuse the agent humanizer's 401/auth detection for a clean message.
		return agent.HumanizeError(err)
	}
	return ""
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

	// Trim the pasted key — leading/trailing whitespace from copy-paste is the
	// single most common cause of "provider rejected the API key" 401s.
	key := strings.TrimSpace(apiKey)
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

// SetupModel configures a model by its full reference, supporting dynamically
// discovered models (via ListProviderModels) and custom model strings the user
// types in — anything not in the static catalog. apiBase/apiKey come from the
// chosen provider; displayName is what shows in the model picker.
func (c *ConfigService) SetupModel(displayName, modelRef, apiBase, apiKey string) error {
	modelRef = strings.TrimSpace(modelRef)
	if modelRef == "" {
		return fmt.Errorf("a model is required")
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = modelRef
	}

	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	newModel := config.ModelConfig{
		ModelName: name,
		Model:     modelRef,
		APIBase:   strings.TrimSpace(apiBase),
		APIKey:    strings.TrimSpace(apiKey),
	}
	for i, m := range cfg.ModelList {
		if m.ModelName == name || m.Model == modelRef {
			cfg.ModelList[i] = newModel
			cfg.Agents.Defaults.ModelName = name
			return saveAndNotify(cfg)
		}
	}
	cfg.ModelList = append(cfg.ModelList, newModel)
	cfg.Agents.Defaults.ModelName = name
	return saveAndNotify(cfg)
}

// QuickSetupChannel enables a channel with its token in one call.
// SetChannelAllowFrom sets a channel's allow-list (the sender IDs/usernames
// permitted to message the agent). Empty list means the channel rejects every
// sender (H1, audit: channels fail closed); pass ["*"] to allow everyone.
func (c *ConfigService) SetChannelAllowFrom(channel string, allowFrom []string) error {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return err
	}
	list := config.FlexibleStringSlice{}
	for _, v := range allowFrom {
		if s := strings.TrimSpace(v); s != "" {
			list = append(list, s)
		}
	}
	switch channel {
	case "telegram":
		cfg.Channels.Telegram.AllowFrom = list
	case "discord":
		cfg.Channels.Discord.AllowFrom = list
	case "slack":
		cfg.Channels.Slack.AllowFrom = list
	case "whatsapp":
		cfg.Channels.WhatsApp.AllowFrom = list
	case "feishu":
		cfg.Channels.Feishu.AllowFrom = list
	case "dingtalk":
		cfg.Channels.DingTalk.AllowFrom = list
	case "line":
		cfg.Channels.LINE.AllowFrom = list
	case "qq":
		cfg.Channels.QQ.AllowFrom = list
	case "onebot":
		cfg.Channels.OneBot.AllowFrom = list
	case "wecom":
		cfg.Channels.WeCom.AllowFrom = list
	case "wecom_app":
		cfg.Channels.WeComApp.AllowFrom = list
	case "maixcam":
		cfg.Channels.MaixCam.AllowFrom = list
	default:
		return fmt.Errorf("unknown channel: %s", channel)
	}
	return saveAndNotify(cfg)
}

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
