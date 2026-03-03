package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Agentx-network/agentx/pkg/wallet"
)

// --- wallet_address tool ---

type WalletAddressTool struct{}

func NewWalletAddressTool() *WalletAddressTool { return &WalletAddressTool{} }

func (t *WalletAddressTool) Name() string { return "wallet_address" }

func (t *WalletAddressTool) Description() string {
	return `Get the agent's BSC wallet address. Call this tool immediately when the user asks about their wallet address, crypto address, or BSC address. Returns the public address only — no private key is ever exposed. No parameters required.`
}

func (t *WalletAddressTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *WalletAddressTool) Execute(_ context.Context, _ map[string]any) *ToolResult {
	addr, err := wallet.GetAddress()
	if err != nil {
		return ErrorResult("No wallet configured. Ask the user to set up a wallet first.")
	}
	return SilentResult(fmt.Sprintf("Wallet address: %s (BSC Mainnet, Chain ID 56)", addr))
}

// --- wallet_balance tool ---

type WalletBalanceTool struct{}

func NewWalletBalanceTool() *WalletBalanceTool { return &WalletBalanceTool{} }

func (t *WalletBalanceTool) Name() string { return "wallet_balance" }

func (t *WalletBalanceTool) Description() string {
	return `Check the agent's BSC wallet balances. Call this tool immediately when the user asks about their balance, funds, how much BNB or tokens they have, or anything related to wallet balances. Returns native BNB and all tracked BEP-20 token balances (USDT, USDC, BUSD, DAI, etc). No parameters required.`
}

func (t *WalletBalanceTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *WalletBalanceTool) Execute(_ context.Context, _ map[string]any) *ToolResult {
	balances, err := wallet.GetAllBalances()
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to fetch balances: %v", err))
	}

	var sb strings.Builder
	sb.WriteString("Wallet Balances (BSC Mainnet):\n")
	for _, b := range balances {
		sb.WriteString(fmt.Sprintf("  %s: %s %s\n", b.Name, b.Balance, b.Symbol))
	}
	return SilentResult(sb.String())
}

// --- wallet_send tool ---

type WalletSendTool struct{}

func NewWalletSendTool() *WalletSendTool { return &WalletSendTool{} }

func (t *WalletSendTool) Name() string { return "wallet_send" }

func (t *WalletSendTool) Description() string {
	return `Send BNB or a BEP-20 token from the agent's BSC wallet. Call this when the user asks to send, transfer, or pay crypto. The private key is handled securely — it is decrypted internally for signing and immediately wiped from memory. Only the transaction hash is returned.

IMPORTANT: Before calling this tool, you MUST first call wallet_balance to check funds, then show the user the recipient, amount, and token, and ask for explicit confirmation. Only call wallet_send after the user confirms.`
}

func (t *WalletSendTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"to": map[string]any{
				"type":        "string",
				"description": "Recipient BSC address (0x...)",
			},
			"amount": map[string]any{
				"type":        "string",
				"description": "Amount to send (e.g. \"0.01\" for 0.01 BNB or \"10.5\" for 10.5 USDT)",
			},
			"token": map[string]any{
				"type":        "string",
				"description": "Token symbol to send. Use \"BNB\" for native BNB, or a token symbol like \"USDT\", \"USDC\", \"BUSD\", \"DAI\". Defaults to BNB if not specified.",
			},
		},
		"required": []string{"to", "amount"},
	}
}

func (t *WalletSendTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	to, _ := args["to"].(string)
	amount, _ := args["amount"].(string)
	token, _ := args["token"].(string)

	if to == "" || amount == "" {
		return ErrorResult("Both 'to' address and 'amount' are required.")
	}

	// Default to BNB
	if token == "" {
		token = "BNB"
	}
	token = strings.ToUpper(token)

	if token == "BNB" {
		txHash, err := wallet.SendBNB(to, amount)
		if err != nil {
			return ErrorResult(fmt.Sprintf("Transaction failed: %v", err))
		}
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Transaction sent! TX hash: %s\nView on BscScan: https://bscscan.com/tx/%s", txHash, txHash),
			ForUser: fmt.Sprintf("Sent %s BNB to %s\nTX: https://bscscan.com/tx/%s", amount, to, txHash),
		}
	}

	// ERC-20 token — find contract from tracked tokens
	balances, err := wallet.GetAllBalances()
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to look up token: %v", err))
	}

	var contract string
	var decimals int
	for _, b := range balances {
		if strings.ToUpper(b.Symbol) == token && b.Contract != "" {
			contract = b.Contract
			decimals = b.Decimals
			break
		}
	}

	if contract == "" {
		return ErrorResult(fmt.Sprintf("Token %s not found in tracked tokens. Add it first via the Wallet page.", token))
	}

	txHash, err := wallet.SendToken(contract, to, amount, decimals)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Transaction failed: %v", err))
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Transaction sent! TX hash: %s\nView on BscScan: https://bscscan.com/tx/%s", txHash, txHash),
		ForUser: fmt.Sprintf("Sent %s %s to %s\nTX: https://bscscan.com/tx/%s", amount, token, to, txHash),
	}
}
