package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Agentx-network/agentx/pkg/wallet"
)

// ERC-8004 IdentityRegistry on BSC mainnet
const identityRegistryAddr = "0x8004A169FB4a3325136EB29fA0ceB6D2e539a432"

// Self-hosted IPFS pin API
const ipfsPinAPI = "https://ipfs.agentx.network/api/pin"
const ipfsPinKey = "a40a3e2c08ecec11731ddbe4decb19594f8d5574d9a4327d"

// RegistryInfo is returned to the frontend.
type RegistryInfo struct {
	Registered bool   `json:"registered"`
	AgentName  string `json:"agentName"`
	AgentID    string `json:"agentId"`
	Address    string `json:"address"`
	Chain      string `json:"chain"`
	Metadata   string `json:"metadata"`
	TxHash     string `json:"txHash"`
	Timestamp  string `json:"timestamp"`
}

type registryFile struct {
	AgentName  string `json:"agent_name"`
	AgentID    string `json:"agent_id"`
	Address    string `json:"address"`
	Chain      string `json:"chain"`
	Metadata   string `json:"metadata"`
	TxHash     string `json:"tx_hash"`
	Timestamp  string `json:"timestamp"`
	Registered bool   `json:"registered"`
}

// RegistryService manages ERC-8004 agent registration.
type RegistryService struct {
	ctx context.Context
}

func NewRegistryService() *RegistryService {
	return &RegistryService{}
}

func (r *RegistryService) startup(ctx context.Context) {
	r.ctx = ctx
}

// GetRegistration returns the current agent registration status.
func (r *RegistryService) GetRegistration() (*RegistryInfo, error) {
	rf, err := loadRegistry()
	if err != nil {
		return &RegistryInfo{Registered: false}, nil
	}
	return &RegistryInfo{
		Registered: rf.Registered,
		AgentName:  rf.AgentName,
		AgentID:    rf.AgentID,
		Address:    rf.Address,
		Chain:      rf.Chain,
		Metadata:   rf.Metadata,
		TxHash:     rf.TxHash,
		Timestamp:  rf.Timestamp,
	}, nil
}

// pinToIPFS uploads JSON metadata to the self-hosted IPFS node and returns the CID.
func pinToIPFS(_ string, metadata []byte) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", ipfsPinAPI, strings.NewReader(string(metadata)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ipfsPinKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("IPFS pin API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("IPFS pin API returned status %d", resp.StatusCode)
	}

	var pinResp struct {
		CID string `json:"cid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pinResp); err != nil {
		return "", fmt.Errorf("failed to parse IPFS pin response: %w", err)
	}
	if pinResp.CID == "" {
		return "", fmt.Errorf("IPFS pin returned empty CID")
	}
	return pinResp.CID, nil
}

// RegisterAgent registers the agent on-chain via the ERC-8004 IdentityRegistry
// contract on BSC. Metadata is pinned to the self-hosted IPFS node, then
// ipfs://{CID} is passed as the agentURI to the contract's register(string) function.
func (r *RegistryService) RegisterAgent(agentName string, metadata string) (*RegistryInfo, error) {
	wi, err := wallet.GetWallet()
	if err != nil {
		return nil, fmt.Errorf("wallet required: generate a wallet first")
	}

	// Build agent metadata JSON
	agentMeta := map[string]interface{}{
		"name":     agentName,
		"platform": "AgentX",
		"chain":    "BSC",
		"address":  wi.Address,
	}
	if metadata != "" {
		var extra map[string]interface{}
		if json.Unmarshal([]byte(metadata), &extra) == nil {
			for k, v := range extra {
				agentMeta[k] = v
			}
		}
	}
	metaJSON, _ := json.Marshal(agentMeta)

	// Pin metadata to IPFS
	cid, err := pinToIPFS(agentName, metaJSON)
	if err != nil {
		return nil, fmt.Errorf("IPFS pinning failed: %w", err)
	}

	agentURI := "ipfs://" + cid

	// ABI-encode the register(string) call
	callData := abiEncodeRegister(agentURI)

	// Get nonce
	nonce, err := getNonce(wi.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := getGasPrice()
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Gas limit for contract interaction (register mints ERC-721, needs ~210k gas)
	gasLimit := uint64(300000)

	// Sign the transaction
	rawTx, err := signTransaction(nonce, gasPrice, gasLimit, identityRegistryAddr, big.NewInt(0), callData)
	if err != nil {
		return nil, fmt.Errorf("transaction signing failed: %w", err)
	}

	// Broadcast the transaction
	txHash, err := sendRawTransaction(rawTx)
	if err != nil {
		return nil, fmt.Errorf("broadcast failed: %w", err)
	}

	// Save registration locally
	rf := registryFile{
		AgentName:  agentName,
		AgentID:    "",
		Address:    wi.Address,
		Chain:      "BSC",
		Metadata:   agentURI,
		TxHash:     txHash,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Registered: true,
	}
	if err := saveRegistry(rf); err != nil {
		return nil, fmt.Errorf("tx sent (%s) but failed to save locally: %w", txHash, err)
	}

	return &RegistryInfo{
		Registered: true,
		AgentName:  rf.AgentName,
		AgentID:    rf.AgentID,
		Address:    rf.Address,
		Chain:      rf.Chain,
		Metadata:   rf.Metadata,
		TxHash:     txHash,
		Timestamp:  rf.Timestamp,
	}, nil
}

// UnregisterAgent clears the local registration record.
func (r *RegistryService) UnregisterAgent() error {
	_ = os.Remove(registryPath())
	return nil
}

// --- BSC JSON-RPC helpers ---

func getNonce(address string) (uint64, error) {
	payload := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_getTransactionCount","params":["%s","latest"],"id":1}`,
		address,
	)
	result, err := bscRPC(payload)
	if err != nil {
		return 0, err
	}
	n := new(big.Int)
	n.SetString(strings.TrimPrefix(result, "0x"), 16)
	return n.Uint64(), nil
}

func getGasPrice() (*big.Int, error) {
	payload := `{"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":1}`
	result, err := bscRPC(payload)
	if err != nil {
		return nil, err
	}
	gp := new(big.Int)
	gp.SetString(strings.TrimPrefix(result, "0x"), 16)
	return gp, nil
}

func sendRawTransaction(signedTx string) (string, error) {
	payload := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["%s"],"id":1}`,
		signedTx,
	)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(
		"https://bsc-dataseed.binance.org/",
		"application/json",
		strings.NewReader(payload),
	)
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

func bscRPC(payload string) (string, error) {
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

func base64Encode(data []byte) string {
	const enc = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var b strings.Builder
	for i := 0; i < len(data); i += 3 {
		var n uint32
		remaining := len(data) - i
		switch {
		case remaining >= 3:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			b.WriteByte(enc[n>>18&0x3f])
			b.WriteByte(enc[n>>12&0x3f])
			b.WriteByte(enc[n>>6&0x3f])
			b.WriteByte(enc[n&0x3f])
		case remaining == 2:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8
			b.WriteByte(enc[n>>18&0x3f])
			b.WriteByte(enc[n>>12&0x3f])
			b.WriteByte(enc[n>>6&0x3f])
			b.WriteByte('=')
		case remaining == 1:
			n = uint32(data[i]) << 16
			b.WriteByte(enc[n>>18&0x3f])
			b.WriteByte(enc[n>>12&0x3f])
			b.WriteByte('=')
			b.WriteByte('=')
		}
	}
	return b.String()
}

// --- file helpers ---

func registryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentx", "registry.json")
}

func loadRegistry() (*registryFile, error) {
	data, err := os.ReadFile(registryPath())
	if err != nil {
		return nil, err
	}
	var rf registryFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, err
	}
	return &rf, nil
}

func saveRegistry(rf registryFile) error {
	p := registryPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}
