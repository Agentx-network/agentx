package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/Agentx-network/agentx/cmd/agentx/internal"
	"github.com/Agentx-network/agentx/pkg/auth"
)

func statusCmd() {
	cfg, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	configPath := internal.GetConfigPath()

	fmt.Printf("%s agentx Status\n", internal.Logo)
	fmt.Printf("Version: %s\n", internal.FormatVersion())
	build, _ := internal.FormatBuildInfo()
	if build != "" {
		fmt.Printf("Build: %s\n", build)
	}
	fmt.Println()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Config:", configPath, "✓")
	} else {
		fmt.Println("Config:", configPath, "✗")
	}

	workspace := cfg.WorkspacePath()
	if _, err := os.Stat(workspace); err == nil {
		fmt.Println("Workspace:", workspace, "✓")
	} else {
		fmt.Println("Workspace:", workspace, "✗")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Model: %s\n", cfg.Agents.Defaults.GetModelName())

		// Show configured models from model_list (primary config)
		if len(cfg.ModelList) > 0 {
			fmt.Println("\nConfigured Models:")
			for _, mc := range cfg.ModelList {
				keyStatus := "✗ no key"
				if mc.APIKey != "" {
					keyStatus = "✓"
				} else if mc.APIBase != "" && (strings.Contains(mc.Model, "ollama") || strings.Contains(mc.Model, "vllm")) {
					keyStatus = "✓ local"
				}
				fmt.Printf("  %s (%s) %s\n", mc.ModelName, mc.Model, keyStatus)
				if mc.APIBase != "" {
					fmt.Printf("    Base: %s\n", mc.APIBase)
				}
			}
		}

		// Also check legacy providers config
		legacyProviders := []struct {
			name   string
			hasKey bool
		}{
			{"OpenRouter", cfg.Providers.OpenRouter.APIKey != ""},
			{"Anthropic", cfg.Providers.Anthropic.APIKey != ""},
			{"OpenAI", cfg.Providers.OpenAI.APIKey != ""},
			{"Gemini", cfg.Providers.Gemini.APIKey != ""},
			{"Groq", cfg.Providers.Groq.APIKey != ""},
			{"DeepSeek", cfg.Providers.DeepSeek.APIKey != ""},
		}
		hasLegacy := false
		for _, lp := range legacyProviders {
			if lp.hasKey {
				hasLegacy = true
				break
			}
		}
		if hasLegacy {
			fmt.Println("\nLegacy Providers:")
			for _, lp := range legacyProviders {
				if lp.hasKey {
					fmt.Printf("  %s: ✓\n", lp.name)
				}
			}
		}

		store, _ := auth.LoadStore()
		if store != nil && len(store.Credentials) > 0 {
			fmt.Println("\nOAuth/Token Auth:")
			for provider, cred := range store.Credentials {
				status := "authenticated"
				if cred.IsExpired() {
					status = "expired"
				} else if cred.NeedsRefresh() {
					status = "needs refresh"
				}
				fmt.Printf("  %s (%s): %s\n", provider, cred.AuthMethod, status)
			}
		}
	}
}
