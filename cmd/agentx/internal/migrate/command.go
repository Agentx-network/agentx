package migrate

import (
	"github.com/spf13/cobra"

	"github.com/Agentx-network/agentx/pkg/migrate"
)

func NewMigrateCommand() *cobra.Command {
	var opts migrate.Options

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from OpenClaw to AgentX",
		Args:  cobra.NoArgs,
		Example: `  agentx migrate
  agentx migrate --dry-run
  agentx migrate --refresh
  agentx migrate --force`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := migrate.Run(opts)
			// Print the summary whenever we have a result (even on partial-failure
			// errors) so the user sees what did and didn't migrate, then surface
			// the error for a non-zero exit.
			if result != nil && !opts.DryRun {
				migrate.PrintSummary(result)
			}
			return err
		},
	}

	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false,
		"Show what would be migrated without making changes")
	cmd.Flags().BoolVar(&opts.Refresh, "refresh", false,
		"Re-sync workspace files from OpenClaw (repeatable)")
	cmd.Flags().BoolVar(&opts.ConfigOnly, "config-only", false,
		"Only migrate config, skip workspace files")
	cmd.Flags().BoolVar(&opts.WorkspaceOnly, "workspace-only", false,
		"Only migrate workspace files, skip config")
	cmd.Flags().BoolVar(&opts.Force, "force", false,
		"Skip confirmation prompts and proceed past unreadable config / partial failures")
	cmd.Flags().StringVar(&opts.OpenClawHome, "openclaw-home", "",
		"Override OpenClaw home directory (default: ~/.openclaw)")
	cmd.Flags().StringVar(&opts.AgentXHome, "agentx-home", "",
		"Override AgentX home directory (default: ~/.agentx)")

	return cmd
}
