import { useState, useEffect, useCallback } from "react";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import NeonInput from "../components/ui/NeonInput";

interface WalletData {
  address: string;
  chain: string;
  createdAt: string;
}

interface TokenBalance {
  symbol: string;
  name: string;
  contract: string;
  balance: string;
  decimals: number;
}

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

const TOKEN_ICONS: Record<string, string> = {
  BNB: "🟡",
  USDT: "🟢",
  USDC: "🔵",
  BUSD: "🟡",
  DAI: "🟠",
};

function truncateAddr(addr: string) {
  if (addr.length <= 14) return addr;
  return addr.slice(0, 8) + "..." + addr.slice(-6);
}

export default function WalletPage({ showToast }: Props) {
  const [wallet, setWallet] = useState<WalletData | null>(null);
  const [balances, setBalances] = useState<TokenBalance[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [copied, setCopied] = useState(false);
  const [showFullAddr, setShowFullAddr] = useState(false);
  const [addingToken, setAddingToken] = useState(false);
  const [newToken, setNewToken] = useState({ symbol: "", name: "", contract: "", decimals: "18" });

  const loadWallet = useCallback(async () => {
    try {
      const w = await window.go.main.WalletService.GetWallet();
      if (w?.address) setWallet(w);
    } catch {
      /* no wallet */
    }
  }, []);

  const loadBalances = useCallback(async () => {
    try {
      const b = await window.go.main.WalletService.GetAllBalances();
      setBalances(b || []);
    } catch {
      /* noop */
    }
  }, []);

  useEffect(() => {
    (async () => {
      await loadWallet();
      await loadBalances();
      setLoading(false);
    })();
  }, [loadWallet, loadBalances]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await loadBalances();
    setRefreshing(false);
    showToast("Balances refreshed", "success");
  };

  const handleGenerate = async () => {
    setLoading(true);
    try {
      const w = await window.go.main.WalletService.GenerateWallet();
      setWallet(w);
      showToast("Wallet generated!", "success");
      await loadBalances();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setLoading(false);
    }
  };

  const copyAddress = () => {
    if (!wallet) return;
    navigator.clipboard.writeText(wallet.address);
    setCopied(true);
    showToast("Address copied!", "success");
    setTimeout(() => setCopied(false), 2000);
  };

  const handleAddToken = async () => {
    try {
      await window.go.main.WalletService.AddToken(
        newToken.symbol,
        newToken.name,
        newToken.contract,
        parseInt(newToken.decimals) || 18,
      );
      showToast(`${newToken.symbol} added!`, "success");
      setNewToken({ symbol: "", name: "", contract: "", decimals: "18" });
      setAddingToken(false);
      await loadBalances();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  const handleRemoveToken = async (contract: string, symbol: string) => {
    try {
      await window.go.main.WalletService.RemoveToken(contract);
      showToast(`${symbol} removed`, "success");
      await loadBalances();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-neon-cyan/50 text-sm uppercase tracking-widest animate-pulse">Loading wallet...</div>
      </div>
    );
  }

  // No wallet yet — show generate prompt
  if (!wallet) {
    return (
      <div className="max-w-lg mx-auto mt-20 space-y-6">
        <div className="text-center space-y-3">
          <div className="w-20 h-20 mx-auto rounded-2xl bg-neon-cyan/5 border border-neon-cyan/20 flex items-center justify-center">
            <span className="text-4xl text-neon-cyan/40 drop-shadow-[0_0_12px_rgba(0,255,255,0.4)]">&#9670;</span>
          </div>
          <h2 className="text-2xl font-bold uppercase tracking-[0.2em] text-glow-pink">Wallet</h2>
          <p className="text-white/35 text-sm">No wallet found. Generate one to enable on-chain features.</p>
        </div>
        <NeonButton onClick={handleGenerate} size="lg" className="w-full">
          Generate Wallet
        </NeonButton>
      </div>
    );
  }

  // Separate native (BNB) and tokens
  const nativeBalance = balances.find((b) => !b.contract);
  const tokenBalances = balances.filter((b) => b.contract);

  return (
    <div className="space-y-5 max-w-3xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold uppercase tracking-[0.2em] text-glow-pink">Wallet</h2>
        <NeonButton onClick={handleRefresh} variant="ghost" size="sm" disabled={refreshing}>
          {refreshing ? "Refreshing..." : "Refresh Balances"}
        </NeonButton>
      </div>

      {/* Wallet card */}
      <NeonCard variant="cyan" glow>
        <div className="space-y-4">
          {/* Header row */}
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-xl bg-neon-cyan/10 border border-neon-cyan/20 flex items-center justify-center text-xl">
              <span className="drop-shadow-[0_0_10px_rgba(0,255,255,0.6)]">&#9670;</span>
            </div>
            <div className="flex-1">
              <div className="text-xs uppercase tracking-widest text-neon-cyan/60 font-bold">Agent Wallet</div>
              <div className="text-xs text-white/30 flex items-center gap-1.5 mt-0.5">
                <div className="w-1.5 h-1.5 rounded-full bg-yellow-400" />
                BSC Mainnet · Chain ID 56
              </div>
            </div>
            <div className="text-[10px] uppercase tracking-widest text-neon-green/70 bg-neon-green/10 border border-neon-green/20 px-2.5 py-1 rounded-full font-bold">
              Active
            </div>
          </div>

          {/* Address */}
          <div
            className="relative group cursor-pointer"
            onMouseEnter={() => setShowFullAddr(true)}
            onMouseLeave={() => setShowFullAddr(false)}
            onClick={copyAddress}
          >
            <div className="bg-black/40 border border-neon-cyan/15 rounded-lg px-4 py-3.5 font-mono text-sm text-neon-cyan/90 tracking-wide hover:border-neon-cyan/40 hover:shadow-[0_0_20px_rgba(0,255,255,0.1)] transition-all flex items-center justify-between">
              <span className="break-all select-all">
                {showFullAddr ? wallet.address : truncateAddr(wallet.address)}
              </span>
              <span className="ml-3 text-[10px] uppercase tracking-widest text-white/30 group-hover:text-neon-cyan/80 transition-colors whitespace-nowrap">
                {copied ? "✓ Copied" : "Click to copy"}
              </span>
            </div>
            {showFullAddr && (
              <div className="absolute left-0 right-0 -bottom-1 translate-y-full z-20 bg-black/95 border border-neon-cyan/20 rounded-lg px-4 py-2.5 font-mono text-xs text-neon-cyan/80 break-all shadow-[0_0_30px_rgba(0,255,255,0.15)] select-all">
                {wallet.address}
              </div>
            )}
          </div>

          {/* Native balance big display */}
          {nativeBalance && (
            <div className="bg-black/30 rounded-lg px-4 py-3 border border-white/[0.04]">
              <div className="text-[10px] uppercase tracking-widest text-white/30 mb-1">Native Balance</div>
              <div className="flex items-baseline gap-2">
                <span className="text-2xl font-bold text-white/90 font-mono">{nativeBalance.balance}</span>
                <span className="text-sm text-yellow-400/70 font-bold">BNB</span>
              </div>
            </div>
          )}

          {/* Meta */}
          <div className="flex items-center justify-between px-1 text-[10px] text-white/25">
            <span>Created {new Date(wallet.createdAt).toLocaleDateString()}</span>
            <span className="font-mono">{wallet.chain}</span>
          </div>
        </div>
      </NeonCard>

      {/* Token balances */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-bold uppercase tracking-widest text-white/50">Token Balances</h3>
          <span className="text-[10px] text-white/25 uppercase tracking-widest">{tokenBalances.length} tokens</span>
        </div>

        <div className="space-y-1.5">
          {tokenBalances.map((tok) => (
            <div
              key={tok.contract}
              className="flex items-center justify-between px-4 py-3 rounded-xl bg-white/[0.02] border border-white/[0.04] hover:border-white/[0.08] hover:bg-white/[0.03] transition-all group"
            >
              <div className="flex items-center gap-3">
                <span className="text-lg">{TOKEN_ICONS[tok.symbol] || "🪙"}</span>
                <div>
                  <div className="text-sm font-bold text-white/80">{tok.symbol}</div>
                  <div className="text-[11px] text-white/30">{tok.name}</div>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-sm font-mono text-white/70">{tok.balance}</span>
                <button
                  onClick={() => handleRemoveToken(tok.contract, tok.symbol)}
                  className="opacity-0 group-hover:opacity-100 text-xs text-red-400/50 hover:text-red-400 px-2 py-1 rounded transition-all"
                  title="Remove token"
                >
                  ✕
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Add token */}
      {!addingToken ? (
        <NeonButton onClick={() => setAddingToken(true)} size="lg" className="w-full">
          + Add Custom Token
        </NeonButton>
      ) : (
        <NeonCard variant="cyan" glow>
          <div className="space-y-3">
            <h3 className="text-sm font-bold uppercase tracking-widest text-neon-cyan">Add Custom Token</h3>
            <p className="text-xs text-white/40">Paste any BSC BEP-20 token contract address to track its balance.</p>
            <div className="grid grid-cols-2 gap-2">
              <NeonInput
                value={newToken.symbol}
                onChange={(v) => setNewToken({ ...newToken, symbol: v })}
                placeholder="Symbol (e.g. CAKE)"
              />
              <NeonInput
                value={newToken.name}
                onChange={(v) => setNewToken({ ...newToken, name: v })}
                placeholder="Name (e.g. PancakeSwap)"
              />
            </div>
            <NeonInput
              value={newToken.contract}
              onChange={(v) => setNewToken({ ...newToken, contract: v })}
              placeholder="Contract address (0x...)"
            />
            <div className="flex gap-2">
              <NeonInput
                value={newToken.decimals}
                onChange={(v) => setNewToken({ ...newToken, decimals: v })}
                placeholder="Decimals (18)"
              />
              <NeonButton onClick={handleAddToken} disabled={!newToken.symbol || !newToken.contract} size="lg" className="flex-1">
                Add Token
              </NeonButton>
              <NeonButton onClick={() => setAddingToken(false)} variant="danger" size="lg">
                Cancel
              </NeonButton>
            </div>
          </div>
        </NeonCard>
      )}
    </div>
  );
}
