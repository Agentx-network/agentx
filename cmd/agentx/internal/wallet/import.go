package wallet

import (
	"fmt"

	"github.com/Agentx-network/agentx/pkg/wallet"
	"github.com/spf13/cobra"
)

func newImportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "import <private-key-hex>",
		Short: "Import a wallet from a hex-encoded private key",
		Long:  "Import an existing BSC wallet by providing a 32-byte hex-encoded private key. If a wallet already exists it will be replaced.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			info, err := wallet.ImportPrivateKey(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Wallet imported\n")
			fmt.Printf("  Address: %s\n", info.Address)
			fmt.Printf("  Chain:   %s (Chain ID 56)\n", info.Chain)
			return nil
		},
	}
}
