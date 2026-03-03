package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Agentx-network/agentx/pkg/wallet"
)

// WalletService manages the agent's on-chain wallet.
// All operations delegate to the shared pkg/wallet package.
type WalletService struct {
	ctx       context.Context
	dashboard *DashboardService
}

func NewWalletService() *WalletService {
	return &WalletService{}
}

func (w *WalletService) startup(ctx context.Context) {
	w.ctx = ctx
}

// SetDashboard wires the dashboard service so we can restart the gateway
// after wallet generation — the gateway must restart to pick up the new wallet.
func (w *WalletService) SetDashboard(d *DashboardService) {
	w.dashboard = d
}

// restartGateway silently restarts the gateway in the background so it picks
// up the newly created/regenerated wallet. Without this, wallet tools in the
// gateway won't work until the user manually restarts.
func (w *WalletService) restartGateway() {
	if w.dashboard == nil {
		return
	}
	go func() {
		_ = w.dashboard.RestartGateway()
	}()
}

// GenerateWallet creates a new wallet or returns the existing one.
func (w *WalletService) GenerateWallet() (*wallet.WalletInfo, error) {
	// If wallet already exists, return it (no restart needed)
	if info, err := wallet.GetInfo(); err == nil {
		return info, nil
	}
	info, err := wallet.GenerateWallet()
	if err != nil {
		return nil, err
	}
	w.restartGateway()
	return info, nil
}

// RegenerateWallet deletes the existing wallet and creates a fresh one.
func (w *WalletService) RegenerateWallet() (*wallet.WalletInfo, error) {
	home, _ := os.UserHomeDir()
	_ = os.Remove(filepath.Join(home, ".agentx", "wallet.json"))
	info, err := wallet.GenerateWallet()
	if err != nil {
		return nil, err
	}
	w.restartGateway()
	return info, nil
}

// ExportPrivateKey decrypts and returns the stored private key as hex.
func (w *WalletService) ExportPrivateKey() (string, error) {
	return wallet.ExportPrivateKey()
}

// ImportPrivateKey imports a wallet from hex-encoded private key.
func (w *WalletService) ImportPrivateKey(hexKey string) (*wallet.WalletInfo, error) {
	info, err := wallet.ImportPrivateKey(hexKey)
	if err != nil {
		return nil, err
	}
	w.restartGateway()
	return info, nil
}

// GetWallet returns the stored wallet info (no private key).
func (w *WalletService) GetWallet() (*wallet.WalletInfo, error) {
	return wallet.GetInfo()
}

// GetAllBalances returns BNB + all tracked token balances.
func (w *WalletService) GetAllBalances() ([]wallet.TokenBalance, error) {
	return wallet.GetAllBalances()
}

// GetBalance returns the native BNB balance string.
func (w *WalletService) GetBalance() (string, error) {
	balances, err := wallet.GetAllBalances()
	if err != nil {
		return "0 BNB", nil
	}
	for _, b := range balances {
		if b.Symbol == "BNB" {
			return b.Balance + " BNB", nil
		}
	}
	return "0 BNB", nil
}

// GetTokens returns the list of tracked tokens.
func (w *WalletService) GetTokens() []wallet.TokenConfig {
	return wallet.GetTokens()
}

// AddToken adds a custom token to track.
func (w *WalletService) AddToken(symbol, name, contract string, decimals int) error {
	return wallet.AddToken(symbol, name, contract, decimals)
}

// RemoveToken removes a tracked token by contract address.
func (w *WalletService) RemoveToken(contract string) error {
	return wallet.RemoveToken(contract)
}
