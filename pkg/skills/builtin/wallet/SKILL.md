---
name: wallet
description: Manage the agent's BSC wallet — check balances and send BNB/tokens securely
inline: true
---

# Wallet Management

You have three wallet tools available. ALWAYS call them directly — do NOT guess or assume the wallet state.

## Rules

1. When the user asks about their wallet address → call `wallet_address` immediately
2. When the user asks about their balance → call `wallet_balance` immediately
3. When the user asks to send funds → call `wallet_balance` first, confirm with user, then call `wallet_send`
4. NEVER say "wallet not configured" or "set up required" without first calling the tool and getting an error back
5. Your private key is never exposed to you. All signing is internal. Never ask the user for their private key.
6. For sending, ALWAYS confirm with the user before executing `wallet_send`

## Tools

### `wallet_address`
Returns the wallet's public BSC address. Call with no parameters.

### `wallet_balance`
Returns BNB + all tracked token balances (USDT, USDC, BUSD, DAI, etc). Call with no parameters.

### `wallet_send`
Sends BNB or a BEP-20 token. Parameters:
- `to` — recipient BSC address (required)
- `amount` — decimal string e.g. "0.01" (required)
- `token` — "BNB" (default) or symbol like "USDT", "USDC"

## Send Flow
1. Call `wallet_balance` to check funds
2. Show user: recipient, amount, token
3. Ask user to confirm
4. Call `wallet_send` only after confirmation
5. Report tx hash and BscScan link: `https://bscscan.com/tx/{hash}`
