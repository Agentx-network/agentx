# Wallet Management

You can manage the user's BSC (Binance Smart Chain) wallet using the `agentx wallet` CLI commands via the `exec` tool.

## Available Commands

| Command | Description |
|---------|-------------|
| `agentx wallet generate` | Generate a new BSC wallet (returns JSON with address, chain, createdAt) |
| `agentx wallet info` | Show wallet address and chain info (JSON) |
| `agentx wallet balance` | Show all balances — BNB + tracked tokens (JSON array) |
| `agentx wallet export` | Export private key as hex string |
| `agentx wallet import <hex-key>` | Import a private key (overwrites existing wallet) |
| `agentx wallet tokens` | List all tracked tokens (JSON array) |
| `agentx wallet add-token --symbol X --name Y --contract Z --decimals N` | Add a custom BEP-20 token to track |
| `agentx wallet remove-token <contract-address>` | Remove a tracked token |

## Usage Guidelines

- Always check if a wallet exists with `agentx wallet info` before generating a new one.
- All output is JSON for easy parsing — do not add extra formatting.
- The `export` command returns a raw hex private key — warn the user about security before exporting.
- The `import` command accepts a hex-encoded secp256k1 private key (with or without 0x prefix) and overwrites any existing wallet.
- Default tracked tokens: USDT, USDC, BUSD, DAI on BSC.
- Balance queries hit the BSC mainnet RPC — they require network connectivity.

## Examples

Generate a wallet:
```
agentx wallet generate
```

Check balances:
```
agentx wallet balance
```

Add a custom token:
```
agentx wallet add-token --symbol CAKE --name "PancakeSwap" --contract 0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82 --decimals 18
```
