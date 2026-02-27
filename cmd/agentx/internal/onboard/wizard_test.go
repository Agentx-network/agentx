package onboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Agentx-network/agentx/pkg/config"
)

func TestFindProvider(t *testing.T) {
	tests := []struct {
		id       string
		wantName string
		wantNil  bool
	}{
		{"openai", "OpenAI", false},
		{"anthropic", "Anthropic", false},
		{"ollama", "Ollama (local)", false},
		{"gemini", "Google Gemini", false},
		{"deepseek", "DeepSeek", false},
		{"groq", "Groq", false},
		{"openrouter", "OpenRouter (100+ models)", false},
		{"mistral", "Mistral AI", false},
		{"cerebras", "Cerebras", false},
		{"nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			p := findProvider(tt.id)
			if tt.wantNil {
				assert.Nil(t, p)
			} else {
				require.NotNil(t, p)
				assert.Equal(t, tt.wantName, p.Name)
				assert.Equal(t, tt.id, p.ID)
				assert.NotEmpty(t, p.ModelName)
				assert.NotEmpty(t, p.Model)
				assert.NotEmpty(t, p.APIBase)
			}
		})
	}
}

func TestBuildModelConfig(t *testing.T) {
	t.Run("with API key", func(t *testing.T) {
		p := findProvider("openai")
		require.NotNil(t, p)

		mc := buildModelConfig(p, "sk-test123")
		assert.Equal(t, "gpt-5.2", mc.ModelName)
		assert.Equal(t, "openai/gpt-5.2", mc.Model)
		assert.Equal(t, "https://api.openai.com/v1", mc.APIBase)
		assert.Equal(t, "sk-test123", mc.APIKey)
	})

	t.Run("with default key (ollama)", func(t *testing.T) {
		p := findProvider("ollama")
		require.NotNil(t, p)

		mc := buildModelConfig(p, "")
		assert.Equal(t, "llama3", mc.ModelName)
		assert.Equal(t, "ollama/llama3", mc.Model)
		assert.Equal(t, "http://localhost:11434/v1", mc.APIBase)
		assert.Equal(t, "ollama", mc.APIKey)
	})

	t.Run("empty key no default", func(t *testing.T) {
		p := findProvider("anthropic")
		require.NotNil(t, p)

		mc := buildModelConfig(p, "")
		assert.Equal(t, "", mc.APIKey)
	})
}

func TestFindChannel(t *testing.T) {
	tests := []struct {
		id      string
		wantNil bool
	}{
		{"telegram", false},
		{"discord", false},
		{"slack", false},
		{"whatsapp", false},
		{"nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			ch := findChannel(tt.id)
			if tt.wantNil {
				assert.Nil(t, ch)
			} else {
				require.NotNil(t, ch)
				assert.Equal(t, tt.id, ch.ID)
				assert.NotEmpty(t, ch.Name)
				assert.NotEmpty(t, ch.TokenFields)
				assert.NotEmpty(t, ch.HelpURL)
			}
		})
	}
}

func TestApplyChannelConfig(t *testing.T) {
	t.Run("telegram", func(t *testing.T) {
		cfg := config.DefaultConfig()
		applyChannelConfig(cfg, "telegram", []string{"123456:ABC-DEF"})
		assert.True(t, cfg.Channels.Telegram.Enabled)
		assert.Equal(t, "123456:ABC-DEF", cfg.Channels.Telegram.Token)
	})

	t.Run("discord", func(t *testing.T) {
		cfg := config.DefaultConfig()
		applyChannelConfig(cfg, "discord", []string{"discord-token"})
		assert.True(t, cfg.Channels.Discord.Enabled)
		assert.Equal(t, "discord-token", cfg.Channels.Discord.Token)
	})

	t.Run("slack", func(t *testing.T) {
		cfg := config.DefaultConfig()
		applyChannelConfig(cfg, "slack", []string{"xoxb-bot", "xapp-app"})
		assert.True(t, cfg.Channels.Slack.Enabled)
		assert.Equal(t, "xoxb-bot", cfg.Channels.Slack.BotToken)
		assert.Equal(t, "xapp-app", cfg.Channels.Slack.AppToken)
	})

	t.Run("whatsapp", func(t *testing.T) {
		cfg := config.DefaultConfig()
		applyChannelConfig(cfg, "whatsapp", []string{"ws://my-bridge:3001"})
		assert.True(t, cfg.Channels.WhatsApp.Enabled)
		assert.Equal(t, "ws://my-bridge:3001", cfg.Channels.WhatsApp.BridgeURL)
	})

	t.Run("unknown channel", func(t *testing.T) {
		cfg := config.DefaultConfig()
		applyChannelConfig(cfg, "unknown", []string{"token"})
		// Should not panic, channels stay disabled
		assert.False(t, cfg.Channels.Telegram.Enabled)
		assert.False(t, cfg.Channels.Discord.Enabled)
	})

	t.Run("empty tokens", func(t *testing.T) {
		cfg := config.DefaultConfig()
		applyChannelConfig(cfg, "telegram", nil)
		assert.True(t, cfg.Channels.Telegram.Enabled)
		assert.Equal(t, "", cfg.Channels.Telegram.Token)
	})
}

func TestIsSystemdAvailable(t *testing.T) {
	// Should return a bool without panicking, regardless of environment
	result := isSystemdAvailable()
	assert.IsType(t, true, result)
}

func TestIsLaunchdAvailable(t *testing.T) {
	result := isLaunchdAvailable()
	assert.IsType(t, true, result)
}

func TestIsServiceAvailable(t *testing.T) {
	// Should return true if either systemd or launchd is available
	result := isServiceAvailable()
	assert.IsType(t, true, result)
	assert.Equal(t, isSystemdAvailable() || isLaunchdAvailable(), result)
}

func TestSaveWizardConfig(t *testing.T) {
	t.Run("basic provider config", func(t *testing.T) {
		p := findProvider("openai")
		require.NotNil(t, p)

		cfg := saveWizardConfig(p, "sk-test", "", nil)

		assert.Len(t, cfg.ModelList, 1)
		assert.Equal(t, "gpt-5.2", cfg.ModelList[0].ModelName)
		assert.Equal(t, "sk-test", cfg.ModelList[0].APIKey)
		assert.Equal(t, "gpt-5.2", cfg.Agents.Defaults.ModelName)
		assert.Empty(t, cfg.Agents.Defaults.Model)
	})

	t.Run("with channel", func(t *testing.T) {
		p := findProvider("anthropic")
		require.NotNil(t, p)

		cfg := saveWizardConfig(p, "sk-ant-test", "telegram", []string{"bot-token"})

		assert.Len(t, cfg.ModelList, 1)
		assert.Equal(t, "claude-sonnet-4.6", cfg.Agents.Defaults.ModelName)
		assert.True(t, cfg.Channels.Telegram.Enabled)
		assert.Equal(t, "bot-token", cfg.Channels.Telegram.Token)
	})

	t.Run("skip channel", func(t *testing.T) {
		p := findProvider("ollama")
		require.NotNil(t, p)

		cfg := saveWizardConfig(p, "", "skip", nil)

		assert.Len(t, cfg.ModelList, 1)
		assert.Equal(t, "ollama", cfg.ModelList[0].APIKey)
		assert.False(t, cfg.Channels.Telegram.Enabled)
		assert.False(t, cfg.Channels.Discord.Enabled)
	})
}
