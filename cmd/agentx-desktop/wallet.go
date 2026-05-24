package main

import (
	"context"

	"github.com/Agentx-network/agentx/pkg/wallet"
)

// Re-export types so Wails bindings stay the same.
type TokenBalance = wallet.TokenBalance
type WalletInfo = wallet.WalletInfo
type TokenConfig = wallet.TokenConfig

// WalletService manages the agent's on-chain wallet.
type WalletService struct {
	ctx context.Context
}

func NewWalletService() *WalletService {
	return &WalletService{}
}

func (w *WalletService) startup(ctx context.Context) {
	w.ctx = ctx
}

func (w *WalletService) GenerateWallet() (*WalletInfo, error) {
	return wallet.GenerateWallet()
}

func (w *WalletService) RegenerateWallet() (*WalletInfo, error) {
	_ = wallet.DeleteWallet()
	return wallet.GenerateWallet()
}

func (w *WalletService) ExportPrivateKey() (string, error) {
	return wallet.ExportPrivateKey()
}

func (w *WalletService) ImportPrivateKey(hexKey string) (*WalletInfo, error) {
	return wallet.ImportPrivateKey(hexKey)
}

func (w *WalletService) GetWallet() (*WalletInfo, error) {
	return wallet.GetWallet()
}

func (w *WalletService) GetAllBalances() ([]TokenBalance, error) {
	return wallet.GetAllBalances()
}

func (w *WalletService) GetBalance() (string, error) {
	return wallet.GetBalance()
}

func (w *WalletService) GetTokens() []TokenConfig {
	return wallet.GetTokens()
}

func (w *WalletService) AddToken(symbol, name, contract string, decimals int) error {
	return wallet.AddToken(symbol, name, contract, decimals)
}

func (w *WalletService) RemoveToken(contract string) error {
	return wallet.RemoveToken(contract)
}
