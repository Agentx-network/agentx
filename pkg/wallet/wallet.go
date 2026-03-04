// Package wallet provides BSC wallet management for AgentX.
// Used by both the desktop app (Wails) and CLI commands.
package wallet

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

// WalletInfo is returned to callers (CLI, desktop).
type WalletInfo struct {
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	CreatedAt string `json:"createdAt"`
}

// TokenBalance is returned per token.
type TokenBalance struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Contract string `json:"contract"`
	Balance  string `json:"balance"`
	Decimals int    `json:"decimals"`
}

// TokenConfig stored in tokens.json.
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

// DefaultTokens are the BSC tokens shown out of the box.
var DefaultTokens = []TokenConfig{
	{Symbol: "USDT", Name: "Tether USD", Contract: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18},
	{Symbol: "USDC", Name: "USD Coin", Contract: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", Decimals: 18},
	{Symbol: "BUSD", Name: "Binance USD", Contract: "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", Decimals: 18},
	{Symbol: "DAI", Name: "Dai Stablecoin", Contract: "0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3", Decimals: 18},
}

const bscRPC = "https://bsc-dataseed.binance.org/"

// GenerateWallet creates a new secp256k1 keypair and stores the encrypted
// private key in ~/.agentx/wallet.json. If a wallet already exists it is
// returned without generating a new one.
func GenerateWallet() (*WalletInfo, error) {
	existing, err := GetWallet()
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

	address := ToChecksumAddress(addrBytes)

	encrypted, err := EncryptKey(privKey.Serialize())
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

	EnsureDefaultTokens()

	return &WalletInfo{
		Address:   address,
		Chain:     "BSC",
		CreatedAt: wf.CreatedAt,
	}, nil
}

// ImportPrivateKey takes a hex-encoded private key, derives the BSC address,
// encrypts the key, and saves a new wallet (overwrites any existing wallet).
func ImportPrivateKey(hexKey string) (*WalletInfo, error) {
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

	address := ToChecksumAddress(addrBytes)

	encrypted, err := EncryptKey(privKey.Serialize())
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

	EnsureDefaultTokens()

	return &WalletInfo{
		Address:   address,
		Chain:     "BSC",
		CreatedAt: wf.CreatedAt,
	}, nil
}

// ExportPrivateKey decrypts and returns the stored private key as a hex string.
func ExportPrivateKey() (string, error) {
	wf, err := loadWallet()
	if err != nil {
		return "", fmt.Errorf("no wallet found: %w", err)
	}

	encrypted, err := hex.DecodeString(wf.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("corrupted wallet data: %w", err)
	}

	privBytes, err := DecryptKey(encrypted)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return hex.EncodeToString(privBytes), nil
}

// GetWallet returns stored wallet info (no private key).
func GetWallet() (*WalletInfo, error) {
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
func GetAllBalances() ([]TokenBalance, error) {
	wf, err := loadWallet()
	if err != nil {
		return nil, err
	}

	EnsureDefaultTokens()
	tokens := LoadTokens()

	var balances []TokenBalance

	// Native BNB
	bnb, _ := QueryBSCBalance(wf.Address)
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
		bal, _ := QueryTokenBalance(wf.Address, tok.Contract, tok.Decimals)
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

// GetBalance returns the native BNB balance string.
func GetBalance() (string, error) {
	wf, err := loadWallet()
	if err != nil {
		return "0 BNB", nil
	}
	balance, err := QueryBSCBalance(wf.Address)
	if err != nil {
		return "0 BNB", nil
	}
	return balance + " BNB", nil
}

// GetTokens returns the list of tracked tokens.
func GetTokens() []TokenConfig {
	EnsureDefaultTokens()
	return LoadTokens()
}

// AddToken adds a custom token to track.
func AddToken(symbol, name, contract string, decimals int) error {
	if symbol == "" || contract == "" {
		return fmt.Errorf("symbol and contract are required")
	}
	if decimals <= 0 {
		decimals = 18
	}

	EnsureDefaultTokens()
	tokens := LoadTokens()

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
func RemoveToken(contract string) error {
	tokens := LoadTokens()
	lower := strings.ToLower(contract)
	var filtered []TokenConfig
	for _, t := range tokens {
		if strings.ToLower(t.Contract) != lower {
			filtered = append(filtered, t)
		}
	}
	return saveTokens(filtered)
}

// DeleteWallet removes the wallet file.
func DeleteWallet() error {
	return os.Remove(WalletPath())
}

// --- Helpers (exported for txsign.go in desktop) ---

// ToChecksumAddress converts raw address bytes to EIP-55 checksummed hex.
func ToChecksumAddress(addr []byte) string {
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

// WalletPath returns ~/.agentx/wallet.json.
func WalletPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentx", "wallet.json")
}

// TokensPath returns ~/.agentx/tokens.json.
func TokensPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentx", "tokens.json")
}

func saveWallet(wf walletFile) error {
	p := WalletPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// LoadEncryptedKey returns the hex-encoded encrypted key from the wallet file.
func LoadEncryptedKey() (string, error) {
	wf, err := loadWallet()
	if err != nil {
		return "", err
	}
	return wf.EncryptedKey, nil
}

func loadWallet() (*walletFile, error) {
	data, err := os.ReadFile(WalletPath())
	if err != nil {
		return nil, err
	}
	var wf walletFile
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, err
	}
	return &wf, nil
}


// EnsureDefaultTokens creates the tokens file with defaults if missing.
func EnsureDefaultTokens() {
	p := TokensPath()
	if _, err := os.Stat(p); err == nil {
		return
	}
	_ = saveTokens(DefaultTokens)
}

// LoadTokens reads the tokens file.
func LoadTokens() []TokenConfig {
	data, err := os.ReadFile(TokensPath())
	if err != nil {
		return DefaultTokens
	}
	var tf tokensFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return DefaultTokens
	}
	return tf.Tokens
}

func saveTokens(tokens []TokenConfig) error {
	p := TokensPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tokensFile{Tokens: tokens}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// DeriveEncryptionKey derives a machine-specific AES-256 key.
func DeriveEncryptionKey() []byte {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	key := sha256.Sum256([]byte(fmt.Sprintf("agentx-wallet:%s:%s", hostname, home)))
	return key[:]
}

// EncryptKey encrypts plaintext with AES-256-GCM using the derived key.
func EncryptKey(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(DeriveEncryptionKey())
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

// DecryptKey decrypts ciphertext with AES-256-GCM using the derived key.
func DecryptKey(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(DeriveEncryptionKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

// QueryBSCBalance queries native BNB balance via BSC RPC.
func QueryBSCBalance(address string) (string, error) {
	payload := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_getBalance","params":["%s","latest"],"id":1}`,
		address,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bscRPC, strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
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

	return FormatWei(result.Result, 18), nil
}

// QueryTokenBalance queries an ERC-20 balanceOf(address).
func QueryTokenBalance(walletAddr, contract string, decimals int) (string, error) {
	addrPadded := fmt.Sprintf("000000000000000000000000%s", strings.TrimPrefix(strings.ToLower(walletAddr), "0x"))
	data := "0x70a08231" + addrPadded

	payload := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"%s","data":"%s"},"latest"],"id":1}`,
		contract, data,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bscRPC, strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
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

	return FormatWei(result.Result, decimals), nil
}

// FormatWei converts a hex wei value to a human-readable decimal string.
func FormatWei(hexVal string, decimals int) string {
	wei := new(big.Int)
	wei.SetString(strings.TrimPrefix(hexVal, "0x"), 16)

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	val := new(big.Float).SetInt(wei)
	val.Quo(val, new(big.Float).SetInt(divisor))

	return val.Text('f', 6)
}
