package wallet

import (
	"fmt"

	"github.com/Agentx-network/agentx/pkg/wallet"
	"github.com/spf13/cobra"
)

func newExportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export the private key (handle with care!)",
		RunE: func(_ *cobra.Command, _ []string) error {
			key, err := wallet.ExportPrivateKey()
			if err != nil {
				return err
			}
			fmt.Println("⚠  Keep this private key safe — anyone with it can control your wallet!")
			fmt.Println(key)
			return nil
		},
	}
}
