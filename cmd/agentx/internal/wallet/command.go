package wallet

import (
	"github.com/spf13/cobra"
)

// NewWalletCommand returns the "agentx wallet" command tree.
func NewWalletCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wallet",
		Aliases: []string{"w"},
		Short:   "Manage the agent's BSC wallet",
	}

	cmd.AddCommand(
		newCreateCommand(),
		newImportCommand(),
		newAddressCommand(),
		newBalanceCommand(),
		newExportCommand(),
	)

	return cmd
}
