package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"

	"github.com/Agentx-network/agentx/pkg/wallet"
)

// BSC mainnet chain ID
const bscChainID = 56

// loadPrivateKey loads and decrypts the private key from wallet.json.
func loadPrivateKey() (*secp256k1.PrivateKey, error) {
	encHex, err := wallet.LoadEncryptedKey()
	if err != nil {
		return nil, fmt.Errorf("no wallet found: %w", err)
	}
	encrypted, err := hex.DecodeString(encHex)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted key: %w", err)
	}
	keyBytes, err := wallet.DecryptKey(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return secp256k1.PrivKeyFromBytes(keyBytes), nil
}

// signTransaction signs a legacy (pre-EIP-1559) transaction with EIP-155.
func signTransaction(
	nonce uint64,
	gasPrice *big.Int,
	gasLimit uint64,
	to string,
	value *big.Int,
	data []byte,
) (string, error) {
	privKey, err := loadPrivateKey()
	if err != nil {
		return "", err
	}

	toBytes, err := hex.DecodeString(strings.TrimPrefix(to, "0x"))
	if err != nil {
		return "", fmt.Errorf("invalid to address: %w", err)
	}

	chainID := big.NewInt(bscChainID)

	// Build the signing payload: RLP([nonce, gasPrice, gasLimit, to, value, data, chainId, 0, 0])
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

	// Sign with secp256k1
	sig := ecdsa.SignCompact(privKey, hash, false)
	// sig[0] = recovery ID + 27, sig[1:33] = R, sig[33:65] = S
	v := int(sig[0]) - 27
	r := new(big.Int).SetBytes(sig[1:33])
	s := new(big.Int).SetBytes(sig[33:65])

	// EIP-155: v = chainId * 2 + 35 + recovery
	vEIP155 := new(big.Int).SetInt64(int64(v) + int64(chainID.Uint64())*2 + 35)

	// Build the final signed transaction
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

// --- ABI encoding helpers ---

// abiEncodeRegister encodes a call to register(string agentURI).
// Method selector: keccak256("register(string)")[:4]
func abiEncodeRegister(agentURI string) []byte {
	// Method ID for register(string)
	methodSig := keccak256([]byte("register(string)"))[:4]

	// ABI encode the string parameter:
	// offset (32 bytes) + length (32 bytes) + padded string data
	uriBytes := []byte(agentURI)
	offset := padLeft(big.NewInt(32).Bytes(), 32)    // offset to string data = 0x20
	length := padLeft(big.NewInt(int64(len(uriBytes))).Bytes(), 32)

	// Pad string to 32-byte boundary
	paddedLen := ((len(uriBytes) + 31) / 32) * 32
	paddedStr := make([]byte, paddedLen)
	copy(paddedStr, uriBytes)

	result := make([]byte, 0, len(methodSig)+len(offset)+len(length)+len(paddedStr))
	result = append(result, methodSig...)
	result = append(result, offset...)
	result = append(result, length...)
	result = append(result, paddedStr...)
	return result
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

// --- Utility helpers ---

func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
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
