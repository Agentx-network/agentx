package onboard

import (
	"fmt"
	"os"
	"runtime"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/Agentx-network/agentx/cmd/agentx/internal"
	"github.com/Agentx-network/agentx/pkg/config"
)

func runWizard() error {
	configPath := internal.GetConfigPath()

	// Check for existing config
	if _, err := os.Stat(configPath); err == nil {
		var overwrite bool
		err := huh.NewConfirm().
			Title("Config already exists at " + configPath).
			Description("Overwrite existing configuration?").
			Affirmative("Yes").
			Negative("No").
			Value(&overwrite).
			WithTheme(neonTheme()).
			Run()
		if err != nil {
			return err
		}
		if !overwrite {
			fmt.Println("Aborted.")
			return nil
		}
	}

	printBanner()

	// Step 1: Pick AI provider
	providerOpts := make([]huh.Option[string], len(providers))
	for i, p := range providers {
		label := p.Name
		if p.KeyURL != "" {
			label += "  (" + p.KeyURL + ")"
		}
		providerOpts[i] = huh.NewOption(label, p.ID)
	}

	var providerID string
	err := huh.NewSelect[string]().
		Title("Choose your AI provider").
		Description("Pick the provider you have an API key for").
		Options(providerOpts...).
		Value(&providerID).
		WithTheme(neonTheme()).
		Run()
	if err != nil {
		return err
	}

	provider := findProvider(providerID)
	if provider == nil {
		return fmt.Errorf("unknown provider: %s", providerID)
	}

	// Step 2: Enter API key (skip for providers that don't need one)
	var apiKey string
	if provider.NeedsKey {
		err := huh.NewInput().
			Title("Enter your " + provider.Name + " API key").
			Description("Get one at: " + provider.KeyURL).
			Placeholder("sk-...").
			EchoMode(huh.EchoModePassword).
			Value(&apiKey).
			WithTheme(neonTheme()).
			Run()
		if err != nil {
			return err
		}
	}

	// Step 3: Pick channel or skip
	channelOpts := []huh.Option[string]{
		huh.NewOption("Skip (set up later)", "skip"),
	}
	for _, ch := range channels {
		channelOpts = append(channelOpts, huh.NewOption(ch.Name, ch.ID))
	}

	var channelID string
	err = huh.NewSelect[string]().
		Title("Connect a messaging channel?").
		Description("Optional: let your agent receive messages").
		Options(channelOpts...).
		Value(&channelID).
		WithTheme(neonTheme()).
		Run()
	if err != nil {
		return err
	}

	// Step 4: Enter channel tokens if selected
	var channelTokens []string
	if channelID != "skip" {
		ch := findChannel(channelID)
		if ch != nil {
			channelTokens = make([]string, len(ch.TokenFields))
			for i, tf := range ch.TokenFields {
				err := huh.NewInput().
					Title(tf.Label).
					Description("See: " + ch.HelpURL).
					Placeholder(tf.Placeholder).
					Value(&channelTokens[i]).
					WithTheme(neonTheme()).
					Run()
				if err != nil {
					return err
				}
			}
		}
	}

	// Build and save config
	cfg := saveWizardConfig(provider, apiKey, channelID, channelTokens)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}

	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)

	// Step 5: Offer background service setup
	var serviceInstalled bool
	if isServiceAvailable() {
		installService := true
		err := huh.NewConfirm().
			Title("Start gateway as a background service?").
			Description("Installs a background service so the gateway auto-starts on login").
			Affirmative("Yes").
			Negative("No").
			Value(&installService).
			WithTheme(neonTheme()).
			Run()
		if err != nil {
			return err
		}
		if installService {
			if err := installGatewayService(); err != nil {
				fmt.Printf("  Warning: could not install service: %v\n", err)
			} else {
				serviceInstalled = true
			}
		}
	}

	printSuccess(provider, configPath, serviceInstalled)
	return nil
}

func runNonInteractive(providerID, apiKey string) error {
	provider := findProvider(providerID)
	if provider == nil {
		return fmt.Errorf("unknown provider: %s\nAvailable: openai, anthropic, gemini, deepseek, groq, openrouter, ollama, mistral, cerebras", providerID)
	}

	if provider.NeedsKey && apiKey == "" {
		return fmt.Errorf("provider %s requires --api-key (get one at %s)", provider.Name, provider.KeyURL)
	}

	configPath := internal.GetConfigPath()
	cfg := saveWizardConfig(provider, apiKey, "", nil)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}

	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)

	// Auto-install gateway service in non-interactive mode
	var serviceInstalled bool
	if isServiceAvailable() {
		if err := installGatewayService(); err != nil {
			fmt.Printf("  Warning: could not install service: %v\n", err)
		} else {
			serviceInstalled = true
		}
	}

	printSuccess(provider, configPath, serviceInstalled)
	return nil
}

func saveWizardConfig(provider *providerInfo, apiKey, channelID string, channelTokens []string) *config.Config {
	cfg := config.DefaultConfig()

	// Replace model_list with only the selected provider
	cfg.ModelList = []config.ModelConfig{
		buildModelConfig(provider, apiKey),
	}

	// Set default model to the selected provider's model
	cfg.Agents.Defaults.ModelName = provider.ModelName
	cfg.Agents.Defaults.Model = ""

	// Apply channel config if selected
	if channelID != "" && channelID != "skip" {
		applyChannelConfig(cfg, channelID, channelTokens)
	}

	return cfg
}

func applyChannelConfig(cfg *config.Config, channelID string, tokens []string) {
	switch channelID {
	case "telegram":
		cfg.Channels.Telegram.Enabled = true
		if len(tokens) > 0 {
			cfg.Channels.Telegram.Token = tokens[0]
		}
	case "discord":
		cfg.Channels.Discord.Enabled = true
		if len(tokens) > 0 {
			cfg.Channels.Discord.Token = tokens[0]
		}
	case "slack":
		cfg.Channels.Slack.Enabled = true
		if len(tokens) > 0 {
			cfg.Channels.Slack.BotToken = tokens[0]
		}
		if len(tokens) > 1 {
			cfg.Channels.Slack.AppToken = tokens[1]
		}
	case "whatsapp":
		cfg.Channels.WhatsApp.Enabled = true
		if len(tokens) > 0 {
			cfg.Channels.WhatsApp.BridgeURL = tokens[0]
		}
	}
}

func printSuccess(provider *providerInfo, configPath string, serviceInstalled bool) {
	pink := lipgloss.NewStyle().Foreground(neonPink).Bold(true)
	cyan := lipgloss.NewStyle().Foreground(neonCyan)
	purple := lipgloss.NewStyle().Foreground(neonPurple)

	fmt.Println()
	fmt.Println(pink.Render("  ✓ AgentX is ready!"))
	fmt.Println()
	fmt.Println(cyan.Render("  Provider: ") + provider.Name)
	fmt.Println(cyan.Render("  Model:    ") + provider.ModelName)
	fmt.Println(cyan.Render("  Config:   ") + configPath)
	if serviceInstalled {
		if runtime.GOOS == "darwin" {
			fmt.Println(cyan.Render("  Service:  ") + "com.agentx.gateway (running)")
		} else {
			fmt.Println(cyan.Render("  Service:  ") + "agentx-gateway.service (running)")
		}
	}
	fmt.Println()
	fmt.Println(purple.Render("  Next steps:"))
	fmt.Println("    agentx agent -m \"Hello!\"")
	if serviceInstalled {
		fmt.Println()
		fmt.Println(purple.Render("  View gateway logs:"))
		if runtime.GOOS == "darwin" {
			fmt.Println("    tail -f ~/.agentx/gateway.log")
		} else {
			fmt.Println("    journalctl --user -u agentx-gateway -f")
		}
	}
	fmt.Println()
}
