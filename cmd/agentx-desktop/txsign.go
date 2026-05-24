package main

import (
	"math/big"

	"github.com/Agentx-network/agentx/pkg/wallet"
)

// signTransaction delegates to the shared wallet package.
func signTransaction(
	nonce uint64,
	gasPrice *big.Int,
	gasLimit uint64,
	to string,
	value *big.Int,
	data []byte,
) (string, error) {
	return wallet.SignTransaction(nonce, gasPrice, gasLimit, to, value, data)
}

// --- ABI encoding helpers ---

// abiEncodeRegister encodes a call to register(string agentURI).
// Method selector: keccak256("register(string)")[:4]
func abiEncodeRegister(agentURI string) []byte {
	// Method ID for register(string)
	methodSig := wallet.Keccak256([]byte("register(string)"))[:4]

	// ABI encode the string parameter:
	// offset (32 bytes) + length (32 bytes) + padded string data
	uriBytes := []byte(agentURI)
	offset := wallet.PadLeft(big.NewInt(32).Bytes(), 32)
	length := wallet.PadLeft(big.NewInt(int64(len(uriBytes))).Bytes(), 32)

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
