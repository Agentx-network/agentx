package wallet

import (
	"fmt"

	"github.com/Agentx-network/agentx/pkg/wallet"
	"github.com/spf13/cobra"
)

func newAddressCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "address",
		Short: "Show the wallet's BSC address",
		RunE: func(_ *cobra.Command, _ []string) error {
			addr, err := wallet.GetAddress()
			if err != nil {
				return fmt.Errorf("no wallet found — run 'agentx wallet create' or 'agentx wallet import' first")
			}
			fmt.Println(addr)
			return nil
		},
	}
}
