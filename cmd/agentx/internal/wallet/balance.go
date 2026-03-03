package wallet

import (
	"fmt"

	"github.com/Agentx-network/agentx/pkg/wallet"
	"github.com/spf13/cobra"
)

func newBalanceCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "balance",
		Short: "Show BNB and token balances",
		RunE: func(_ *cobra.Command, _ []string) error {
			balances, err := wallet.GetAllBalances()
			if err != nil {
				return fmt.Errorf("failed to fetch balances: %w", err)
			}
			fmt.Println("Wallet Balances (BSC Mainnet):")
			for _, b := range balances {
				fmt.Printf("  %s: %s %s\n", b.Name, b.Balance, b.Symbol)
			}
			return nil
		},
	}
}
