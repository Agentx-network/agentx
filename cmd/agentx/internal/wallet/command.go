package wallet

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Agentx-network/agentx/pkg/wallet"
)

func NewWalletCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wallet",
		Short: "Manage BSC wallet",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newGenerateCmd(),
		newInfoCmd(),
		newBalanceCmd(),
		newExportCmd(),
		newImportCmd(),
		newTokensCmd(),
		newAddTokenCmd(),
		newRemoveTokenCmd(),
		newSendCmd(),
	)

	return cmd
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func newGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate a new BSC wallet",
		RunE: func(_ *cobra.Command, _ []string) error {
			info, err := wallet.GenerateWallet()
			if err != nil {
				return err
			}
			printJSON(info)
			return nil
		},
	}
}

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show wallet address and chain",
		RunE: func(_ *cobra.Command, _ []string) error {
			info, err := wallet.GetWallet()
			if err != nil {
				return fmt.Errorf("no wallet found — run 'agentx wallet generate' first")
			}
			printJSON(info)
			return nil
		},
	}
}

func newBalanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "balance",
		Short: "Show all balances (BNB + tracked tokens)",
		RunE: func(_ *cobra.Command, _ []string) error {
			balances, err := wallet.GetAllBalances()
			if err != nil {
				return fmt.Errorf("no wallet found — run 'agentx wallet generate' first")
			}
			printJSON(balances)
			return nil
		},
	}
}

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export private key (hex)",
		RunE: func(_ *cobra.Command, _ []string) error {
			key, err := wallet.ExportPrivateKey()
			if err != nil {
				return err
			}
			fmt.Println(key)
			return nil
		},
	}
}

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <hex-private-key>",
		Short: "Import a private key",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			info, err := wallet.ImportPrivateKey(args[0])
			if err != nil {
				return err
			}
			printJSON(info)
			return nil
		},
	}
}

func newTokensCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tokens",
		Short: "List tracked tokens",
		RunE: func(_ *cobra.Command, _ []string) error {
			tokens := wallet.GetTokens()
			printJSON(tokens)
			return nil
		},
	}
}

func newAddTokenCmd() *cobra.Command {
	var symbol, name, contract string
	var decimals int

	cmd := &cobra.Command{
		Use:   "add-token",
		Short: "Add a custom token to track",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := wallet.AddToken(symbol, name, contract, decimals); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Token %s added\n", symbol)
			return nil
		},
	}

	cmd.Flags().StringVar(&symbol, "symbol", "", "Token symbol (required)")
	cmd.Flags().StringVar(&name, "name", "", "Token name")
	cmd.Flags().StringVar(&contract, "contract", "", "Contract address (required)")
	cmd.Flags().IntVar(&decimals, "decimals", 18, "Token decimals")
	_ = cmd.MarkFlagRequired("symbol")
	_ = cmd.MarkFlagRequired("contract")

	return cmd
}

func newRemoveTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-token <contract-address>",
		Short: "Remove a tracked token",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := wallet.RemoveToken(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Token removed")
			return nil
		},
	}
}

func newSendCmd() *cobra.Command {
	var token string

	cmd := &cobra.Command{
		Use:   "send <to-address> <amount>",
		Short: "Send BNB or ERC-20 tokens to an address",
		Long: `Send BNB or ERC-20 tokens to a BSC address.

Examples:
  agentx wallet send 0x1234...abcd 0.1              # Send 0.1 BNB
  agentx wallet send 0x1234...abcd 10 --token USDT   # Send 10 USDT`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			toAddress := args[0]
			amount := args[1]

			var result *wallet.SendResult
			var err error

			if token == "" {
				result, err = wallet.SendBNB(toAddress, amount)
			} else {
				result, err = wallet.SendToken(toAddress, amount, token)
			}
			if err != nil {
				return err
			}

			printJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Token symbol to send (e.g. USDT, USDC). Omit for native BNB")

	return cmd
}
