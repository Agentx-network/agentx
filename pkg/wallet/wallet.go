// Package wallet provides secure wallet operations for the AgentX agent.
// The private key never leaves this package — all signing is internal.
package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"
)

const bscRPCURL = "https://bsc-dataseed.binance.org/"
const bscChainID = 56

// WalletInfo is public wallet data (no private key).
type WalletInfo struct {
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	CreatedAt string `json:"createdAt"`
}

// TokenConfig describes a tracked token.
type TokenConfig struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Contract string `json:"contract"`
	Decimals int    `json:"decimals"`
}

// TokenBalance holds a token's balance.
type TokenBalance struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Contract string `json:"contract"`
	Balance  string `json:"balance"`
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

var defaultTokens = []TokenConfig{
	{Symbol: "USDT", Name: "Tether USD", Contract: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18},
	{Symbol: "USDC", Name: "USD Coin", Contract: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", Decimals: 18},
	{Symbol: "BUSD", Name: "Binance USD", Contract: "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", Decimals: 18},
	{Symbol: "DAI", Name: "Dai Stablecoin", Contract: "0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3", Decimals: 18},
}

// --- Public API (safe for agent tools) ---

// GetAddress returns the wallet's public address. No private key is exposed.
func GetAddress() (string, error) {
	wf, err := loadWallet()
	if err != nil {
		return "", fmt.Errorf("no wallet found")
	}
	return wf.Address, nil
}

// GetInfo returns public wallet info.
func GetInfo() (*WalletInfo, error) {
	wf, err := loadWallet()
	if err != nil {
		return nil, fmt.Errorf("no wallet found")
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
		return nil, fmt.Errorf("no wallet found")
	}

	tokens := loadTokens()
	var balances []TokenBalance

	// Native BNB
	bnb, _ := queryBalance(wf.Address)
	if bnb == "" {
		bnb = "0"
	}
	balances = append(balances, TokenBalance{
		Symbol: "BNB", Name: "BNB", Contract: "", Balance: bnb, Decimals: 18,
	})

	// ERC-20 tokens
	for _, tok := range tokens {
		bal, _ := queryTokenBalance(wf.Address, tok.Contract, tok.Decimals)
		if bal == "" {
			bal = "0"
		}
		balances = append(balances, TokenBalance{
			Symbol: tok.Symbol, Name: tok.Name, Contract: tok.Contract,
			Balance: bal, Decimals: tok.Decimals,
		})
	}

	return balances, nil
}

// SendBNB sends native BNB to a recipient. The private key is decrypted
// internally, used for signing, and immediately discarded. The agent never
// sees the key — only the transaction hash is returned.
func SendBNB(to string, amountBNB string) (string, error) {
	if !isValidAddress(to) {
		return "", fmt.Errorf("invalid recipient address")
	}

	wf, err := loadWallet()
	if err != nil {
		return "", fmt.Errorf("no wallet found")
	}

	// Parse amount to wei
	weiValue, err := bnbToWei(amountBNB)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}

	nonce, err := getNonce(wf.Address)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := getGasPrice()
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	gasLimit := uint64(21000) // standard transfer

	rawTx, err := signTransaction(wf.EncryptedKey, nonce, gasPrice, gasLimit, to, weiValue, nil)
	if err != nil {
		return "", fmt.Errorf("signing failed: %w", err)
	}

	txHash, err := sendRawTransaction(rawTx)
	if err != nil {
		return "", fmt.Errorf("broadcast failed: %w", err)
	}

	return txHash, nil
}

// SendToken sends an ERC-20 token to a recipient. Same security model as SendBNB.
func SendToken(contractAddr, to string, amount string, decimals int) (string, error) {
	if !isValidAddress(to) {
		return "", fmt.Errorf("invalid recipient address")
	}
	if !isValidAddress(contractAddr) {
		return "", fmt.Errorf("invalid contract address")
	}

	wf, err := loadWallet()
	if err != nil {
		return "", fmt.Errorf("no wallet found")
	}

	// Parse amount to token units
	tokenWei, err := parseTokenAmount(amount, decimals)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %w", err)
	}

	// Build ERC-20 transfer(address,uint256) calldata
	data := buildTransferData(to, tokenWei)

	nonce, err := getNonce(wf.Address)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := getGasPrice()
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	gasLimit := uint64(60000) // ERC-20 transfer

	rawTx, err := signTransaction(wf.EncryptedKey, nonce, gasPrice, gasLimit, contractAddr, big.NewInt(0), data)
	if err != nil {
		return "", fmt.Errorf("signing failed: %w", err)
	}

	txHash, err := sendRawTransaction(rawTx)
	if err != nil {
		return "", fmt.Errorf("broadcast failed: %w", err)
	}

	return txHash, nil
}

// --- Internal: signing (private key never leaves here) ---

func signTransaction(encryptedKeyHex string, nonce uint64, gasPrice *big.Int, gasLimit uint64, to string, value *big.Int, data []byte) (string, error) {
	encrypted, err := hex.DecodeString(encryptedKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid encrypted key")
	}

	keyBytes, err := decryptKey(encrypted)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}
	defer zeroBytes(keyBytes) // wipe key from memory

	privKey := secp256k1.PrivKeyFromBytes(keyBytes)

	toBytes, err := hex.DecodeString(strings.TrimPrefix(to, "0x"))
	if err != nil {
		return "", fmt.Errorf("invalid to address")
	}

	chainID := big.NewInt(bscChainID)

	sigPayload := rlpEncode([]interface{}{
		bigToMinBytes(new(big.Int).SetUint64(nonce)),
		bigToMinBytes(gasPrice),
		bigToMinBytes(new(big.Int).SetUint64(gasLimit)),
		toBytes,
		bigToMinBytes(value),
		data,
		bigToMinBytes(chainID),
		[]byte{},
		[]byte{},
	})

	hash := keccak256(sigPayload)
	sig := ecdsa.SignCompact(privKey, hash, false)
	v := int(sig[0]) - 27
	r := new(big.Int).SetBytes(sig[1:33])
	s := new(big.Int).SetBytes(sig[33:65])

	vEIP155 := new(big.Int).SetInt64(int64(v) + int64(chainID.Uint64())*2 + 35)

	signedTx := rlpEncode([]interface{}{
		bigToMinBytes(new(big.Int).SetUint64(nonce)),
		bigToMinBytes(gasPrice),
		bigToMinBytes(new(big.Int).SetUint64(gasLimit)),
		toBytes,
		bigToMinBytes(value),
		data,
		bigToMinBytes(vEIP155),
		bigToMinBytes(r),
		bigToMinBytes(s),
	})

	return "0x" + hex.EncodeToString(signedTx), nil
}

// zeroBytes wipes a byte slice to remove key material from memory.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// --- Internal: BSC RPC ---

func bscRPC(method string, params []interface{}) (json.RawMessage, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	data, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(bscRPCURL, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", result.Error.Message)
	}
	return result.Result, nil
}

func queryBalance(address string) (string, error) {
	raw, err := bscRPC("eth_getBalance", []interface{}{address, "latest"})
	if err != nil {
		return "", err
	}
	var hexVal string
	if err := json.Unmarshal(raw, &hexVal); err != nil {
		return "", err
	}
	if hexVal == "" || hexVal == "0x0" {
		return "0", nil
	}
	return formatWei(hexVal, 18), nil
}

func queryTokenBalance(wallet, contract string, decimals int) (string, error) {
	addrPadded := fmt.Sprintf("000000000000000000000000%s", strings.TrimPrefix(strings.ToLower(wallet), "0x"))
	callData := "0x70a08231" + addrPadded

	raw, err := bscRPC("eth_call", []interface{}{
		map[string]string{"to": contract, "data": callData},
		"latest",
	})
	if err != nil {
		return "", err
	}
	var hexVal string
	if err := json.Unmarshal(raw, &hexVal); err != nil {
		return "", err
	}
	if hexVal == "" || hexVal == "0x" || hexVal == "0x0" {
		return "0", nil
	}
	return formatWei(hexVal, decimals), nil
}

func getNonce(address string) (uint64, error) {
	raw, err := bscRPC("eth_getTransactionCount", []interface{}{address, "latest"})
	if err != nil {
		return 0, err
	}
	var hexVal string
	if err := json.Unmarshal(raw, &hexVal); err != nil {
		return 0, err
	}
	n := new(big.Int)
	n.SetString(strings.TrimPrefix(hexVal, "0x"), 16)
	return n.Uint64(), nil
}

func getGasPrice() (*big.Int, error) {
	raw, err := bscRPC("eth_gasPrice", nil)
	if err != nil {
		return nil, err
	}
	var hexVal string
	if err := json.Unmarshal(raw, &hexVal); err != nil {
		return nil, err
	}
	price := new(big.Int)
	price.SetString(strings.TrimPrefix(hexVal, "0x"), 16)
	return price, nil
}

func sendRawTransaction(rawTx string) (string, error) {
	raw, err := bscRPC("eth_sendRawTransaction", []interface{}{rawTx})
	if err != nil {
		return "", err
	}
	var txHash string
	if err := json.Unmarshal(raw, &txHash); err != nil {
		return "", err
	}
	return txHash, nil
}

// --- Internal: file I/O ---

func walletPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentx", "wallet.json")
}

func tokensPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentx", "tokens.json")
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

// --- Internal: crypto ---

func deriveEncryptionKey() []byte {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	key := sha256.Sum256([]byte(fmt.Sprintf("agentx-wallet:%s:%s", hostname, home)))
	return key[:]
}

func decryptKey(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveEncryptionKey())
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

// --- Internal: encoding helpers ---

func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

func formatWei(hexVal string, decimals int) string {
	wei := new(big.Int)
	wei.SetString(strings.TrimPrefix(hexVal, "0x"), 16)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	val := new(big.Float).SetInt(wei)
	val.Quo(val, new(big.Float).SetInt(divisor))
	return val.Text('f', 6)
}

func bnbToWei(amountBNB string) (*big.Int, error) {
	f, _, err := new(big.Float).Parse(amountBNB, 10)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", amountBNB)
	}
	decimals := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	f.Mul(f, decimals)
	wei, _ := f.Int(nil)
	return wei, nil
}

func parseTokenAmount(amount string, decimals int) (*big.Int, error) {
	f, _, err := new(big.Float).Parse(amount, 10)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", amount)
	}
	dec := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	f.Mul(f, dec)
	wei, _ := f.Int(nil)
	return wei, nil
}

func isValidAddress(addr string) bool {
	addr = strings.TrimPrefix(addr, "0x")
	if len(addr) != 40 {
		return false
	}
	_, err := hex.DecodeString(addr)
	return err == nil
}

// buildTransferData builds ERC-20 transfer(address,uint256) calldata.
func buildTransferData(to string, amount *big.Int) []byte {
	// transfer(address,uint256) selector: 0xa9059cbb
	methodID := keccak256([]byte("transfer(address,uint256)"))[:4]
	toAddr, _ := hex.DecodeString(strings.TrimPrefix(to, "0x"))
	paddedTo := padLeft(toAddr, 32)
	paddedAmt := padLeft(amount.Bytes(), 32)

	result := make([]byte, 0, 4+32+32)
	result = append(result, methodID...)
	result = append(result, paddedTo...)
	result = append(result, paddedAmt...)
	return result
}

func bigToMinBytes(b *big.Int) []byte {
	if b.Sign() == 0 {
		return []byte{}
	}
	return b.Bytes()
}

func padLeft(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	padded := make([]byte, size)
	copy(padded[size-len(b):], b)
	return padded
}

// --- RLP encoding ---

func rlpEncode(items []interface{}) []byte {
	var payload []byte
	for _, item := range items {
		payload = append(payload, rlpEncodeItem(item)...)
	}
	return rlpEncodeLength(len(payload), 0xc0, payload)
}

func rlpEncodeItem(item interface{}) []byte {
	switch v := item.(type) {
	case []byte:
		if len(v) == 1 && v[0] < 0x80 {
			return v
		}
		if len(v) == 0 {
			return []byte{0x80}
		}
		return rlpEncodeLength(len(v), 0x80, v)
	case []interface{}:
		return rlpEncode(v)
	default:
		return []byte{0x80}
	}
}

func rlpEncodeLength(dataLen int, offset byte, payload []byte) []byte {
	if dataLen < 56 {
		result := make([]byte, 1+len(payload))
		result[0] = offset + byte(dataLen)
		copy(result[1:], payload)
		return result
	}
	lenBytes := bigToMinBytes(big.NewInt(int64(dataLen)))
	result := make([]byte, 1+len(lenBytes)+len(payload))
	result[0] = offset + 55 + byte(len(lenBytes))
	copy(result[1:], lenBytes)
	copy(result[1+len(lenBytes):], payload)
	return result
}
