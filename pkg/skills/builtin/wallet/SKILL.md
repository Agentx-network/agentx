---
name: wallet
description: Manage the agent's BSC wallet — check balances and send BNB/tokens securely
---

# Wallet Management

Securely manage the agent's BSC (BNB Smart Chain) wallet. You have three wallet tools available.

## Security Model

- Your wallet's **private key is never exposed** to you. All signing happens internally.
- You can freely check the address and balances — these are read-only operations.
- For **sending transactions**, you MUST always confirm with the user first before executing.
- Never ask the user for their private key. You do not need it — signing is handled securely by the system.

## Available Tools

### `wallet_address`
Returns the wallet's public BSC address. Use this when:
- The user asks "what's my wallet address?"
- You need to share the address for receiving funds
- Verifying which wallet is active

### `wallet_balance`
Returns all token balances (BNB + USDT, USDC, BUSD, DAI, and any custom tokens). Use when:
- The user asks about their balance
- Before sending a transaction (to verify sufficient funds)
- Periodic balance checks

### `wallet_send`
Sends BNB or a BEP-20 token. Parameters:
- `to` — recipient BSC address (required)
- `amount` — amount to send as a decimal string, e.g. "0.01" (required)
- `token` — "BNB" for native, or a symbol like "USDT", "USDC" (optional, defaults to BNB)

**CRITICAL: Always follow this flow before sending:**
1. Check the wallet balance first to ensure sufficient funds
2. Clearly show the user: recipient address, amount, and token
3. Ask the user to explicitly confirm: "Send X TOKEN to 0x...?"
4. Only execute `wallet_send` after user confirmation
5. Report the transaction hash and BscScan link after success

## Examples

**User: "What's my balance?"**
→ Use `wallet_balance`, then format the results clearly.

**User: "Send 0.01 BNB to 0xABC..."**
→ First use `wallet_balance` to check funds, then confirm with the user, then use `wallet_send`.

**User: "Send 10 USDT to 0xABC..."**
→ Use `wallet_balance`, confirm sufficient USDT, confirm with user, use `wallet_send` with token="USDT".

## Important Notes
- All transactions are on **BSC Mainnet** (Chain ID 56)
- Gas fees are paid in BNB — ensure there's enough BNB for gas even when sending tokens
- Transaction links: `https://bscscan.com/tx/{hash}`
- If no wallet exists, tell the user to set one up via the desktop app's Wallet page
