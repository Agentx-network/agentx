package skills

import (
	"github.com/spf13/cobra"

	"github.com/Agentx-network/agentx/pkg/config"
)

func newSearchCommand(cfgFn func() (*config.Config, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search available skills",
		Args:  cobra.ExactArgs(1),
		Example: `  agentx skills search web
  agentx skills search "test runner"
  agentx skills search code`,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := cfgFn()
			if err != nil {
				return err
			}
			skillsSearchCmd(cfg, args[0])
			return nil
		},
	}

	return cmd
}
