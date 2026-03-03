package wallet

import (
	"fmt"

	"github.com/Agentx-network/agentx/pkg/wallet"
	"github.com/spf13/cobra"
)

func newCreateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Generate a new BSC wallet",
		RunE: func(_ *cobra.Command, _ []string) error {
			info, err := wallet.GenerateWallet()
			if err != nil {
				return err
			}
			fmt.Printf("✓ Wallet created\n")
			fmt.Printf("  Address: %s\n", info.Address)
			fmt.Printf("  Chain:   %s (Chain ID 56)\n", info.Chain)
			fmt.Println("\nYour private key is encrypted and stored in ~/.agentx/wallet.json")
			fmt.Println("Use 'agentx wallet export' to back up your key.")
			return nil
		},
	}
}
