package onboard

import (
	"embed"

	"github.com/spf13/cobra"
)

//go:generate cp -r ../../../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

func NewOnboardCommand() *cobra.Command {
	var provider string
	var apiKey string

	cmd := &cobra.Command{
		Use:     "onboard",
		Aliases: []string{"o"},
		Short:   "Initialize agentx configuration and workspace",
		Long: `Interactive wizard to set up AgentX with your preferred AI provider and
optional messaging channel. Produces a clean config with only your chosen provider.

Run without flags for the interactive TUI wizard, or use --provider (and --api-key)
for scripted/non-interactive setup.`,
		Example: `  agentx onboard                                    # interactive wizard
  agentx onboard --provider ollama                  # non-interactive, no key needed
  agentx onboard --provider openai --api-key sk-... # non-interactive with key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if provider != "" {
				return runNonInteractive(provider, apiKey)
			}
			return runWizard()
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "AI provider ID (openai, anthropic, gemini, deepseek, groq, openrouter, ollama, mistral, cerebras)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for the selected provider")

	return cmd
}
