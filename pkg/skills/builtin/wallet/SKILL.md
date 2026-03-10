---
name: wallet
description: Manage BSC wallet — generate, import, export, check balances, track tokens, send funds
---

# Wallet Management

You can manage the user's BSC (Binance Smart Chain) wallet using the `agentx wallet` CLI commands via the `exec` tool.

## Available Commands

| Command | Description |
|---------|-------------|
| `agentx wallet generate` | Generate a new BSC wallet (returns JSON with address, chain, createdAt) |
| `agentx wallet info` | Show wallet address and chain info (JSON) |
| `agentx wallet balance` | Show all balances — BNB + tracked tokens (JSON array) |
| `agentx wallet send <to-address> <amount>` | Send native BNB to an address |
| `agentx wallet send <to-address> <amount> --token SYMBOL` | Send an ERC-20 token (e.g. USDT, USDC) |
| `agentx wallet export` | Export private key as hex string |
| `agentx wallet import <hex-key>` | Import a private key (overwrites existing wallet) |
| `agentx wallet tokens` | List all tracked tokens (JSON array) |
| `agentx wallet add-token --symbol X --name Y --contract Z --decimals N` | Add a custom BEP-20 token to track |
| `agentx wallet remove-token <contract-address>` | Remove a tracked token |

## Usage Guidelines

- Always check if a wallet exists with `agentx wallet info` before generating a new one.
- All output is JSON for easy parsing — do not add extra formatting.
- **Before sending funds**, always confirm the recipient address, amount, and token with the user. Never send funds without explicit user approval.
- The `send` command returns JSON with txHash, from, to, amount, and token fields.
- The amount for `send` is a human-readable decimal (e.g. "0.1" for 0.1 BNB, "10.5" for 10.5 USDT).
- When sending ERC-20 tokens, the token must be in the tracked tokens list. Use `agentx wallet tokens` to check.
- The `export` command returns a raw hex private key — warn the user about security before exporting.
- The `import` command accepts a hex-encoded secp256k1 private key (with or without 0x prefix) and overwrites any existing wallet.
- Default tracked tokens: USDT, USDC, BUSD, DAI on BSC.
- Balance and send operations hit the BSC mainnet RPC — they require network connectivity.
- Sending requires sufficient BNB for gas fees, even when sending ERC-20 tokens.

## Examples

Generate a wallet:
```
agentx wallet generate
```

Check balances:
```
agentx wallet balance
```

Send 0.1 BNB:
```
agentx wallet send 0x1234567890abcdef1234567890abcdef12345678 0.1
```

Send 10 USDT:
```
agentx wallet send 0x1234567890abcdef1234567890abcdef12345678 10 --token USDT
```

Add a custom token:
```
agentx wallet add-token --symbol CAKE --name "PancakeSwap" --contract 0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82 --decimals 18
```
