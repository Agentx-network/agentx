package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/sha3"
)

// TokenBalance is returned per token.
type TokenBalance struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Contract string `json:"contract"` // empty for native BNB
	Balance  string `json:"balance"`
	Decimals int    `json:"decimals"`
}

// WalletInfo is returned to the frontend.
type WalletInfo struct {
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	CreatedAt string `json:"createdAt"`
}

// TokenConfig stored in tokens.json
type TokenConfig struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Contract string `json:"contract"`
	Decimals int    `json:"decimals"`
}

type walletFile struct {
	Address      string `json:"address"`
	EncryptedKey string `json:"encrypted_key"`
	Chain        string `json:"chain"`
	CreatedAt    string `json:"created_at"`
}

type tokensFile struct {
	Tokens []TokenConfig `json:"tokens"`
}

// Default BSC tokens shown out of the box.
var defaultTokens = []TokenConfig{
	{Symbol: "USDT", Name: "Tether USD", Contract: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18},
	{Symbol: "USDC", Name: "USD Coin", Contract: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", Decimals: 18},
	{Symbol: "BUSD", Name: "Binance USD", Contract: "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", Decimals: 18},
	{Symbol: "DAI", Name: "Dai Stablecoin", Contract: "0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3", Decimals: 18},
}

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

// GenerateWallet creates a new secp256k1 keypair and stores the encrypted
// private key in ~/.agentx/wallet.json. If a wallet already exists it is
// returned without generating a new one.
func (w *WalletService) GenerateWallet() (*WalletInfo, error) {
	existing, err := w.GetWallet()
	if err == nil && existing.Address != "" {
		return existing, nil
	}

	privKey, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("key generation failed: %w", err)
	}

	pubBytes := privKey.PubKey().SerializeUncompressed()

	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubBytes[1:])
	addrBytes := hash.Sum(nil)[12:]

	address := toChecksumAddress(addrBytes)

	encrypted, err := encryptKey(privKey.Serialize())
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	wf := walletFile{
		Address:      address,
		EncryptedKey: hex.EncodeToString(encrypted),
		Chain:        "bsc",
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := saveWallet(wf); err != nil {
		return nil, err
	}

	// Ensure default tokens file exists
	ensureDefaultTokens()

	return &WalletInfo{
		Address:   address,
		Chain:     "BSC",
		CreatedAt: wf.CreatedAt,
	}, nil
}

// RegenerateWallet deletes the existing wallet and creates a fresh one.
func (w *WalletService) RegenerateWallet() (*WalletInfo, error) {
	_ = os.Remove(walletPath())
	return w.GenerateWallet()
}

// ExportPrivateKey decrypts and returns the stored private key as a hex string.
// This is used during the uninstall export flow so the user can back up their key.
func (w *WalletService) ExportPrivateKey() (string, error) {
	wf, err := loadWallet()
	if err != nil {
		return "", fmt.Errorf("no wallet found: %w", err)
	}

	encrypted, err := hex.DecodeString(wf.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("corrupted wallet data: %w", err)
	}

	privBytes, err := decryptKey(encrypted)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return hex.EncodeToString(privBytes), nil
}

// ImportPrivateKey takes a hex-encoded private key, derives the BSC address,
// encrypts the key with the current machine's encryption key, and saves a new wallet.
// If a wallet already exists it is overwritten.
func (w *WalletService) ImportPrivateKey(hexKey string) (*WalletInfo, error) {
	privBytes, err := hex.DecodeString(strings.TrimPrefix(hexKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}

	if len(privBytes) != 32 {
		return nil, fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(privBytes))
	}

	privKey := secp256k1.PrivKeyFromBytes(privBytes)
	pubBytes := privKey.PubKey().SerializeUncompressed()

	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubBytes[1:])
	addrBytes := hash.Sum(nil)[12:]

	address := toChecksumAddress(addrBytes)

	encrypted, err := encryptKey(privKey.Serialize())
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	wf := walletFile{
		Address:      address,
		EncryptedKey: hex.EncodeToString(encrypted),
		Chain:        "bsc",
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := saveWallet(wf); err != nil {
		return nil, err
	}

	ensureDefaultTokens()

	return &WalletInfo{
		Address:   address,
		Chain:     "BSC",
		CreatedAt: wf.CreatedAt,
	}, nil
}

// GetWallet returns the stored wallet information (no private key).
func (w *WalletService) GetWallet() (*WalletInfo, error) {
	wf, err := loadWallet()
	if err != nil {
		return nil, err
	}
	return &WalletInfo{
		Address:   wf.Address,
		Chain:     strings.ToUpper(wf.Chain),
		CreatedAt: wf.CreatedAt,
	}, nil
}

// GetAllBalances returns BNB + all tracked token balances.
func (w *WalletService) GetAllBalances() ([]TokenBalance, error) {
	wf, err := loadWallet()
	if err != nil {
		return nil, err
	}

	ensureDefaultTokens()
	tokens := loadTokens()

	var balances []TokenBalance

	// Native BNB
	bnb, _ := queryBSCBalance(wf.Address)
	if bnb == "" {
		bnb = "0"
	}
	balances = append(balances, TokenBalance{
		Symbol:   "BNB",
		Name:     "BNB",
		Contract: "",
		Balance:  bnb,
		Decimals: 18,
	})

	// ERC-20 tokens
	for _, tok := range tokens {
		bal, _ := queryTokenBalance(wf.Address, tok.Contract, tok.Decimals)
		if bal == "" {
			bal = "0"
		}
		balances = append(balances, TokenBalance{
			Symbol:   tok.Symbol,
			Name:     tok.Name,
			Contract: tok.Contract,
			Balance:  bal,
			Decimals: tok.Decimals,
		})
	}

	return balances, nil
}

// GetBalance returns the native BNB balance (kept for backward compat).
func (w *WalletService) GetBalance() (string, error) {
	wf, err := loadWallet()
	if err != nil {
		return "0 BNB", nil
	}
	balance, err := queryBSCBalance(wf.Address)
	if err != nil {
		return "0 BNB", nil
	}
	return balance + " BNB", nil
}

// GetTokens returns the list of tracked tokens.
func (w *WalletService) GetTokens() []TokenConfig {
	ensureDefaultTokens()
	return loadTokens()
}

// AddToken adds a custom token to track.
func (w *WalletService) AddToken(symbol, name, contract string, decimals int) error {
	if symbol == "" || contract == "" {
		return fmt.Errorf("symbol and contract are required")
	}
	if decimals <= 0 {
		decimals = 18
	}

	ensureDefaultTokens()
	tokens := loadTokens()

	// Check for duplicate
	lower := strings.ToLower(contract)
	for _, t := range tokens {
		if strings.ToLower(t.Contract) == lower {
			return fmt.Errorf("token %s already tracked", t.Symbol)
		}
	}

	tokens = append(tokens, TokenConfig{
		Symbol:   strings.ToUpper(symbol),
		Name:     name,
		Contract: contract,
		Decimals: decimals,
	})

	return saveTokens(tokens)
}

// RemoveToken removes a tracked token by contract address.
func (w *WalletService) RemoveToken(contract string) error {
	tokens := loadTokens()
	lower := strings.ToLower(contract)
	var filtered []TokenConfig
	for _, t := range tokens {
		if strings.ToLower(t.Contract) != lower {
			filtered = append(filtered, t)
		}
	}
	return saveTokens(filtered)
}

// --- helpers ---

func toChecksumAddress(addr []byte) string {
	hexAddr := hex.EncodeToString(addr)

	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(hexAddr))
	hashBytes := hash.Sum(nil)

	var b strings.Builder
	b.WriteString("0x")
	for i, c := range hexAddr {
		nibble := hashBytes[i/2]
		if i%2 == 0 {
			nibble >>= 4
		} else {
			nibble &= 0x0f
		}
		if c >= 'a' && c <= 'f' && nibble >= 8 {
			b.WriteByte(byte(c) - 32)
		} else {
			b.WriteByte(byte(c))
		}
	}
	return b.String()
}

func walletPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentx", "wallet.json")
}

func tokensPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentx", "tokens.json")
}

func saveWallet(wf walletFile) error {
	p := walletPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func loadWallet() (*walletFile, error) {
	data, err := os.ReadFile(walletPath())
	if err != nil {
		return nil, err
	}
	var wf walletFile
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

func ensureDefaultTokens() {
	p := tokensPath()
	if _, err := os.Stat(p); err == nil {
		return // already exists
	}
	_ = saveTokens(defaultTokens)
}

func loadTokens() []TokenConfig {
	data, err := os.ReadFile(tokensPath())
	if err != nil {
		return defaultTokens
	}
	var tf tokensFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return defaultTokens
	}
	return tf.Tokens
}

func saveTokens(tokens []TokenConfig) error {
	p := tokensPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tokensFile{Tokens: tokens}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func deriveEncryptionKey() []byte {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	key := sha256.Sum256([]byte(fmt.Sprintf("agentx-wallet:%s:%s", hostname, home)))
	return key[:]
}

func encryptKey(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveEncryptionKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func queryBSCBalance(address string) (string, error) {
	payload := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_getBalance","params":["%s","latest"],"id":1}`,
		address,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(
		"https://bsc-dataseed.binance.org/",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Result == "" || result.Result == "0x0" {
		return "0", nil
	}

	return formatWei(result.Result, 18), nil
}

// queryTokenBalance uses the ERC-20 balanceOf(address) call.
// Method sig: 0x70a08231 + address padded to 32 bytes.
func queryTokenBalance(wallet, contract string, decimals int) (string, error) {
	addrPadded := fmt.Sprintf("000000000000000000000000%s", strings.TrimPrefix(strings.ToLower(wallet), "0x"))
	data := "0x70a08231" + addrPadded

	payload := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"%s","data":"%s"},"latest"],"id":1}`,
		contract, data,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(
		"https://bsc-dataseed.binance.org/",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Result == "" || result.Result == "0x" || result.Result == "0x0" {
		return "0", nil
	}

	return formatWei(result.Result, decimals), nil
}

func formatWei(hexVal string, decimals int) string {
	wei := new(big.Int)
	wei.SetString(strings.TrimPrefix(hexVal, "0x"), 16)

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	val := new(big.Float).SetInt(wei)
	val.Quo(val, new(big.Float).SetInt(divisor))

	return val.Text('f', 6)
}
