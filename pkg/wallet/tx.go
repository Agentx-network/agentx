package wallet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"
)

// BSC mainnet chain ID
const bscChainID = 56

// SendResult is returned after a successful send.
type SendResult struct {
	TxHash string `json:"txHash"`
	From   string `json:"from"`
	To     string `json:"to"`
	Amount string `json:"amount"`
	Token  string `json:"token"`
}

// SendBNB sends native BNB to the given address.
// amount is a decimal string like "0.1".
func SendBNB(toAddress string, amount string) (*SendResult, error) {
	if err := validateAddress(toAddress); err != nil {
		return nil, err
	}

	wei, err := parseDecimalToWei(amount, 18)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	if wei.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	wi, err := GetWallet()
	if err != nil {
		return nil, fmt.Errorf("no wallet found: %w", err)
	}

	nonce, err := GetNonce(wi.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := GetGasPrice()
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Standard gas limit for BNB transfer
	gasLimit := uint64(21000)

	rawTx, err := SignTransaction(nonce, gasPrice, gasLimit, toAddress, wei, nil)
	if err != nil {
		return nil, fmt.Errorf("signing failed: %w", err)
	}

	txHash, err := SendRawTransaction(rawTx)
	if err != nil {
		return nil, fmt.Errorf("broadcast failed: %w", err)
	}

	return &SendResult{
		TxHash: txHash,
		From:   wi.Address,
		To:     toAddress,
		Amount: amount,
		Token:  "BNB",
	}, nil
}

// SendToken sends an ERC-20 token to the given address.
// amount is a decimal string like "10.5". tokenSymbol is used to look up
// the contract address and decimals from the tracked tokens list.
func SendToken(toAddress string, amount string, tokenSymbol string) (*SendResult, error) {
	if err := validateAddress(toAddress); err != nil {
		return nil, err
	}

	// Find token in tracked list
	tokens := LoadTokens()
	var token *TokenConfig
	upperSymbol := strings.ToUpper(tokenSymbol)
	for _, t := range tokens {
		if strings.ToUpper(t.Symbol) == upperSymbol {
			token = &t
			break
		}
	}
	if token == nil {
		return nil, fmt.Errorf("token %q not found in tracked tokens — add it with 'wallet add-token'", tokenSymbol)
	}

	wei, err := parseDecimalToWei(amount, token.Decimals)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	if wei.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	wi, err := GetWallet()
	if err != nil {
		return nil, fmt.Errorf("no wallet found: %w", err)
	}

	// ABI-encode transfer(address,uint256)
	callData := abiEncodeTransfer(toAddress, wei)

	nonce, err := GetNonce(wi.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := GetGasPrice()
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Gas limit for ERC-20 transfer
	gasLimit := uint64(60000)

	rawTx, err := SignTransaction(nonce, gasPrice, gasLimit, token.Contract, big.NewInt(0), callData)
	if err != nil {
		return nil, fmt.Errorf("signing failed: %w", err)
	}

	txHash, err := SendRawTransaction(rawTx)
	if err != nil {
		return nil, fmt.Errorf("broadcast failed: %w", err)
	}

	return &SendResult{
		TxHash: txHash,
		From:   wi.Address,
		To:     toAddress,
		Amount: amount,
		Token:  token.Symbol,
	}, nil
}

// --- Transaction signing ---

// LoadPrivateKey loads and decrypts the private key from wallet.json.
func LoadPrivateKey() (*secp256k1.PrivateKey, error) {
	encHex, err := LoadEncryptedKey()
	if err != nil {
		return nil, fmt.Errorf("no wallet found: %w", err)
	}
	encrypted, err := hex.DecodeString(encHex)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted key: %w", err)
	}
	keyBytes, err := DecryptKey(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return secp256k1.PrivKeyFromBytes(keyBytes), nil
}

// SignTransaction signs a legacy (pre-EIP-1559) transaction with EIP-155.
func SignTransaction(
	nonce uint64,
	gasPrice *big.Int,
	gasLimit uint64,
	to string,
	value *big.Int,
	data []byte,
) (string, error) {
	privKey, err := LoadPrivateKey()
	if err != nil {
		return "", err
	}

	toBytes, err := hex.DecodeString(strings.TrimPrefix(to, "0x"))
	if err != nil {
		return "", fmt.Errorf("invalid to address: %w", err)
	}

	chainID := big.NewInt(bscChainID)

	// Build the signing payload: RLP([nonce, gasPrice, gasLimit, to, value, data, chainId, 0, 0])
	sigPayload := RLPEncode([]interface{}{
		BigToMinBytes(new(big.Int).SetUint64(nonce)),
		BigToMinBytes(gasPrice),
		BigToMinBytes(new(big.Int).SetUint64(gasLimit)),
		toBytes,
		BigToMinBytes(value),
		data,
		BigToMinBytes(chainID),
		[]byte{},
		[]byte{},
	})

	hash := Keccak256(sigPayload)

	// Sign with secp256k1
	sig := ecdsa.SignCompact(privKey, hash, false)
	// sig[0] = recovery ID + 27, sig[1:33] = R, sig[33:65] = S
	v := int(sig[0]) - 27
	r := new(big.Int).SetBytes(sig[1:33])
	s := new(big.Int).SetBytes(sig[33:65])

	// EIP-155: v = chainId * 2 + 35 + recovery
	vEIP155 := new(big.Int).SetInt64(int64(v) + int64(chainID.Uint64())*2 + 35)

	// Build the final signed transaction
	signedTx := RLPEncode([]interface{}{
		BigToMinBytes(new(big.Int).SetUint64(nonce)),
		BigToMinBytes(gasPrice),
		BigToMinBytes(new(big.Int).SetUint64(gasLimit)),
		toBytes,
		BigToMinBytes(value),
		data,
		BigToMinBytes(vEIP155),
		BigToMinBytes(r),
		BigToMinBytes(s),
	})

	return "0x" + hex.EncodeToString(signedTx), nil
}

// --- ABI encoding ---

// abiEncodeTransfer encodes a call to transfer(address,uint256).
func abiEncodeTransfer(to string, amount *big.Int) []byte {
	// Method ID for transfer(address,uint256)
	methodSig := Keccak256([]byte("transfer(address,uint256)"))[:4]

	// ABI encode: address (32 bytes, left-padded) + uint256 (32 bytes, left-padded)
	addrBytes, _ := hex.DecodeString(strings.TrimPrefix(strings.ToLower(to), "0x"))
	paddedAddr := PadLeft(addrBytes, 32)
	paddedAmount := PadLeft(amount.Bytes(), 32)

	result := make([]byte, 0, 4+32+32)
	result = append(result, methodSig...)
	result = append(result, paddedAddr...)
	result = append(result, paddedAmount...)
	return result
}

// --- RLP encoding ---

// RLPEncode encodes a list of items using RLP encoding.
func RLPEncode(items []interface{}) []byte {
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
		return RLPEncode(v)
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
	lenBytes := BigToMinBytes(big.NewInt(int64(dataLen)))
	result := make([]byte, 1+len(lenBytes)+len(payload))
	result[0] = offset + 55 + byte(len(lenBytes))
	copy(result[1:], lenBytes)
	copy(result[1+len(lenBytes):], payload)
	return result
}

// --- BSC JSON-RPC helpers ---

// GetNonce returns the transaction count for the given address.
func GetNonce(address string) (uint64, error) {
	payload := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_getTransactionCount","params":["%s","latest"],"id":1}`,
		address,
	)
	result, err := bscRPCCall(payload)
	if err != nil {
		return 0, err
	}
	n := new(big.Int)
	n.SetString(strings.TrimPrefix(result, "0x"), 16)
	return n.Uint64(), nil
}

// GetGasPrice returns the current gas price from BSC.
func GetGasPrice() (*big.Int, error) {
	payload := `{"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":1}`
	result, err := bscRPCCall(payload)
	if err != nil {
		return nil, err
	}
	gp := new(big.Int)
	gp.SetString(strings.TrimPrefix(result, "0x"), 16)
	return gp, nil
}

// SendRawTransaction broadcasts a signed transaction to BSC.
func SendRawTransaction(signedTx string) (string, error) {
	payload := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["%s"],"id":1}`,
		signedTx,
	)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(bscRPC, "application/json", strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return "", err
	}
	if rpcResp.Error != nil {
		return "", fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if rpcResp.Result == "" {
		return "", fmt.Errorf("empty transaction hash returned")
	}
	return rpcResp.Result, nil
}

func bscRPCCall(payload string) (string, error) {
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

	var rpcResp struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return "", err
	}
	if rpcResp.Error != nil {
		return "", fmt.Errorf("RPC: %s", rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

// --- Utility helpers ---

// Keccak256 returns the Keccak-256 hash of the data.
func Keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

// BigToMinBytes converts a big.Int to its minimal byte representation.
func BigToMinBytes(b *big.Int) []byte {
	if b.Sign() == 0 {
		return []byte{}
	}
	return b.Bytes()
}

// PadLeft left-pads a byte slice to the given size.
func PadLeft(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	padded := make([]byte, size)
	copy(padded[size-len(b):], b)
	return padded
}

// validateAddress checks that a BSC/Ethereum address is valid.
func validateAddress(addr string) error {
	if !strings.HasPrefix(addr, "0x") && !strings.HasPrefix(addr, "0X") {
		return fmt.Errorf("address must start with 0x")
	}
	clean := strings.TrimPrefix(strings.TrimPrefix(addr, "0x"), "0X")
	if len(clean) != 40 {
		return fmt.Errorf("address must be 42 characters (0x + 40 hex chars)")
	}
	if _, err := hex.DecodeString(clean); err != nil {
		return fmt.Errorf("address contains invalid hex characters")
	}
	return nil
}

// parseDecimalToWei converts a decimal string (e.g. "1.5") to wei with the given decimals.
func parseDecimalToWei(amount string, decimals int) (*big.Int, error) {
	// Split on decimal point
	parts := strings.SplitN(amount, ".", 2)
	if len(parts) == 1 {
		parts = append(parts, "")
	}

	intPart := parts[0]
	fracPart := parts[1]

	// Truncate or pad fractional part to match decimals
	if len(fracPart) > decimals {
		fracPart = fracPart[:decimals]
	}
	for len(fracPart) < decimals {
		fracPart += "0"
	}

	combined := intPart + fracPart
	// Remove leading zeros but keep at least "0"
	combined = strings.TrimLeft(combined, "0")
	if combined == "" {
		combined = "0"
	}

	wei, ok := new(big.Int).SetString(combined, 10)
	if !ok {
		return nil, fmt.Errorf("invalid number: %s", amount)
	}
	return wei, nil
}
