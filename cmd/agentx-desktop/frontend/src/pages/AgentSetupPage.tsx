import { useState, useEffect, useRef, useCallback } from "react";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import NeonInput from "../components/ui/NeonInput";

interface BootstrapFile {
  name: string;
  path: string;
  content: string;
  exists: boolean;
}

interface SkillEntry {
  name: string;
  source: string;
  description: string;
  path: string;
}

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
  onComplete?: () => void;
}

const fileDescriptions: Record<string, string> = {
  "SOUL.md": "Personality traits, communication style, tone, and values",
  "IDENTITY.md": "Name, tagline, purpose, capabilities, and limitations",
  "AGENTS.md": "Workflow rules, task patterns, and multi-channel behavior",
  "USER.md": "Your profile, preferences, interests, and project context",
};

const HEX = "0123456789abcdef";
const PHASES = [
  "INITIALIZING CRYPTOGRAPHIC ENGINE",
  "GENERATING SECURE ENTROPY",
  "DERIVING SECP256K1 KEYPAIR",
  "COMPUTING KECCAK-256 HASH",
  "ENCODING EIP-55 CHECKSUM",
  "WALLET GENERATED",
];

const STEPS = [
  { key: "wallet", label: "Wallet", num: 1 },
  { key: "identity", label: "Identity", num: 2 },
  { key: "skills", label: "Skills", num: 3 },
] as const;

const TOKEN_ICONS: Record<string, string> = {
  BNB: "🟡",
  USDT: "🟢",
  USDC: "🔵",
  BUSD: "🟡",
  DAI: "🟠",
};

function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}

function randomHex(len: number) {
  let s = "";
  for (let i = 0; i < len; i++) s += HEX[Math.floor(Math.random() * 16)];
  return s;
}

function truncateAddr(addr: string) {
  if (addr.length <= 14) return addr;
  return addr.slice(0, 8) + "..." + addr.slice(-6);
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Step 1: Wallet Generation
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

function WalletStep({
  showToast,
  onNext,
}: {
  showToast: Props["showToast"];
  onNext: () => void;
}) {
  const [wallet, setWallet] = useState<WalletData | null>(null);
  const [balances, setBalances] = useState<TokenBalance[]>([]);
  const [generating, setGenerating] = useState(false);
  const [phase, setPhase] = useState(-1);
  const [displayAddr, setDisplayAddr] = useState("");
  const [revealCount, setRevealCount] = useState(0);
  const [entropyLines, setEntropyLines] = useState<string[]>([]);
  const [copied, setCopied] = useState(false);
  const [showFullAddr, setShowFullAddr] = useState(false);
  const [autoStarted, setAutoStarted] = useState(false);
  const [addingToken, setAddingToken] = useState(false);
  const [newToken, setNewToken] = useState({ symbol: "", name: "", contract: "", decimals: "18" });
  const [importMode, setImportMode] = useState(false);
  const [importKey, setImportKey] = useState("");
  const [importing, setImporting] = useState(false);
  const entropyRef = useRef<number | null>(null);
  const realAddrRef = useRef("");

  useEffect(() => {
    (async () => {
      try {
        const w = await window.go.main.WalletService.GetWallet();
        if (w?.address) {
          setWallet(w);
          setDisplayAddr(w.address);
          loadBalances();
        }
      } catch {
        /* no wallet yet */
      }
    })();
  }, []);

  const loadBalances = async () => {
    try {
      const b = await window.go.main.WalletService.GetAllBalances();
      setBalances(b || []);
    } catch {
      /* noop */
    }
  };

  useEffect(() => {
    if (!wallet && !generating && !autoStarted && !importMode) {
      setAutoStarted(true);
      const t = setTimeout(() => handleGenerate(), 600);
      return () => clearTimeout(t);
    }
  }, [wallet, importMode]);

  const startEntropy = useCallback(() => {
    entropyRef.current = window.setInterval(() => {
      setEntropyLines((prev) => {
        const next = [...prev, randomHex(40)];
        return next.length > 8 ? next.slice(-8) : next;
      });
    }, 120);
  }, []);

  const stopEntropy = useCallback(() => {
    if (entropyRef.current) {
      clearInterval(entropyRef.current);
      entropyRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (phase !== 4) return;
    const addr = realAddrRef.current;
    if (!addr) return;
    let frame = 0;
    const interval = setInterval(() => {
      const revealed = Math.min(frame, addr.length);
      setRevealCount(revealed);
      let display = addr.slice(0, revealed);
      for (let i = revealed; i < addr.length; i++) {
        if (i < 2) display += addr[i];
        else display += HEX[Math.floor(Math.random() * 16)];
      }
      setDisplayAddr(display);
      frame++;
      if (revealed >= addr.length) {
        clearInterval(interval);
        setDisplayAddr(addr);
      }
    }, 55);
    return () => clearInterval(interval);
  }, [phase]);

  const handleGenerate = async () => {
    setGenerating(true);
    setPhase(0);
    setEntropyLines([]);
    setRevealCount(0);
    setDisplayAddr("");
    const walletPromise = window.go.main.WalletService.GenerateWallet();
    await sleep(900);
    setPhase(1);
    startEntropy();
    await sleep(1200);
    setPhase(2);
    await sleep(1100);
    setPhase(3);
    await sleep(900);
    stopEntropy();
    setPhase(4);
    let result: WalletData;
    try {
      result = await walletPromise;
    } catch (e: any) {
      showToast(`Wallet generation failed: ${e}`, "error");
      setGenerating(false);
      setPhase(-1);
      setAutoStarted(false);
      stopEntropy();
      return;
    }
    realAddrRef.current = result.address;
    await sleep(result.address.length * 55 + 400);
    setPhase(5);
    setWallet(result);
    setDisplayAddr(result.address);
    setGenerating(false);
    loadBalances();
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
      loadBalances();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  const handleRemoveToken = async (contract: string, symbol: string) => {
    try {
      await window.go.main.WalletService.RemoveToken(contract);
      showToast(`${symbol} removed`, "success");
      loadBalances();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  const handleImportKey = async () => {
    if (!importKey.trim()) return;
    setImporting(true);
    try {
      const result = await window.go.main.WalletService.ImportPrivateKey(importKey.trim());
      setWallet(result);
      setDisplayAddr(result.address);
      showToast("Wallet imported!", "success");
      setImportMode(false);
      setImportKey("");
      loadBalances();
    } catch (e: any) {
      showToast(`Import failed: ${e}`, "error");
    } finally {
      setImporting(false);
    }
  };

  // ── Wallet exists — show details ──
  if (wallet && !generating) {
    return (
      <div className="space-y-4">
        <NeonCard variant="cyan" glow>
          <div className="space-y-4">
            {/* Header */}
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-neon-cyan/10 border border-neon-cyan/20 flex items-center justify-center text-lg">
                <span className="drop-shadow-[0_0_8px_rgba(0,255,255,0.6)]">&#9670;</span>
              </div>
              <div className="flex-1">
                <div className="text-[11px] uppercase tracking-widest text-neon-cyan/60 font-bold">Agent Wallet</div>
                <div className="text-[11px] text-white/30 flex items-center gap-1.5">
                  <div className="w-1.5 h-1.5 rounded-full bg-yellow-400" />
                  BSC Mainnet
                </div>
              </div>
              <div className="text-[10px] uppercase tracking-widest text-neon-green/70 bg-neon-green/10 border border-neon-green/20 px-2.5 py-1 rounded-full font-bold">
                Active
              </div>
            </div>

            {/* Address — clickable, copyable, hover to see full */}
            <div
              className="relative group cursor-pointer"
              onMouseEnter={() => setShowFullAddr(true)}
              onMouseLeave={() => setShowFullAddr(false)}
              onClick={copyAddress}
            >
              <div className="bg-black/40 border border-neon-cyan/15 rounded-lg px-4 py-3 font-mono text-sm text-neon-cyan/90 tracking-wide hover:border-neon-cyan/40 hover:shadow-[0_0_20px_rgba(0,255,255,0.1)] transition-all flex items-center justify-between">
                <span className="break-all select-all">
                  {showFullAddr ? wallet.address : truncateAddr(wallet.address)}
                </span>
                <span className="ml-3 text-[10px] uppercase tracking-widest text-white/30 group-hover:text-neon-cyan/80 transition-colors whitespace-nowrap">
                  {copied ? "✓ Copied" : "Click to copy"}
                </span>
              </div>
              {/* Full address tooltip on hover */}
              {showFullAddr && (
                <div className="absolute left-0 right-0 -bottom-1 translate-y-full z-20 bg-black/95 border border-neon-cyan/20 rounded-lg px-4 py-2.5 font-mono text-xs text-neon-cyan/80 break-all shadow-[0_0_30px_rgba(0,255,255,0.15)] select-all">
                  {wallet.address}
                </div>
              )}
            </div>

            {/* Token balances */}
            <div className="space-y-1">
              <div className="text-[10px] uppercase tracking-widest text-white/35 px-1 mb-2 font-bold">
                Token Balances
              </div>
              {balances.map((tok) => (
                <div
                  key={tok.symbol + tok.contract}
                  className="flex items-center justify-between px-3 py-2 rounded-lg bg-black/20 border border-white/[0.04] hover:border-white/[0.08] transition-colors group/tok"
                >
                  <div className="flex items-center gap-2.5">
                    <span className="text-sm">{TOKEN_ICONS[tok.symbol] || "🪙"}</span>
                    <div>
                      <div className="text-xs font-bold text-white/70">{tok.symbol}</div>
                      <div className="text-[10px] text-white/25">{tok.name}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-mono text-white/60">{tok.balance}</span>
                    {tok.contract && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleRemoveToken(tok.contract, tok.symbol);
                        }}
                        className="opacity-0 group-hover/tok:opacity-100 text-[10px] text-red-400/60 hover:text-red-400 transition-all"
                        title="Remove token"
                      >
                        ✕
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>

            {/* Add custom token */}
            {!addingToken ? (
              <button
                onClick={() => setAddingToken(true)}
                className="w-full text-[11px] uppercase tracking-widest text-white/25 hover:text-neon-cyan/60 py-2 border border-dashed border-white/10 hover:border-neon-cyan/20 rounded-lg transition-all"
              >
                + Add Custom Token
              </button>
            ) : (
              <div className="space-y-2 p-3 rounded-lg border border-neon-cyan/15 bg-black/20">
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
                  <NeonButton
                    onClick={handleAddToken}
                    disabled={!newToken.symbol || !newToken.contract}
                    size="sm"
                  >
                    Add
                  </NeonButton>
                  <NeonButton
                    onClick={() => setAddingToken(false)}
                    variant="ghost"
                    size="sm"
                  >
                    Cancel
                  </NeonButton>
                </div>
              </div>
            )}

            {/* Meta row */}
            <div className="flex items-center justify-between px-1 pt-1 border-t border-white/[0.04]">
              <div className="text-[10px] text-white/25">
                Created {new Date(wallet.createdAt).toLocaleDateString()}
              </div>
              <div className="text-[10px] text-white/25 font-mono">
                BSC Chain ID: 56
              </div>
            </div>
          </div>
        </NeonCard>

        <NeonButton onClick={onNext} size="lg" className="w-full">
          Next: Set Up Identity &rarr;
        </NeonButton>
      </div>
    );
  }

  // ── Import mode ──
  if (importMode && !generating) {
    return (
      <div className="space-y-4">
        <NeonCard variant="cyan" glow>
          <div className="space-y-4 py-2">
            <div className="text-center space-y-2">
              <div className="w-14 h-14 mx-auto rounded-2xl bg-neon-cyan/10 border border-neon-cyan/20 flex items-center justify-center">
                <span className="text-2xl text-neon-cyan/60 drop-shadow-[0_0_8px_rgba(0,255,255,0.6)]">&#9670;</span>
              </div>
              <h3 className="text-sm font-bold uppercase tracking-widest text-white/70">Import Existing Key</h3>
              <p className="text-xs text-white/35 max-w-xs mx-auto">
                Paste your hex-encoded private key to restore a previously exported wallet.
              </p>
            </div>

            <div className="space-y-3">
              <NeonInput
                value={importKey}
                onChange={setImportKey}
                placeholder="Private key (hex, 64 characters)"
              />
              <div className="flex gap-2">
                <NeonButton
                  onClick={handleImportKey}
                  disabled={!importKey.trim() || importing}
                  size="lg"
                  className="flex-1"
                >
                  {importing ? "Importing..." : "Import Wallet"}
                </NeonButton>
                <NeonButton
                  onClick={() => { setImportMode(false); setImportKey(""); setAutoStarted(false); }}
                  variant="ghost"
                  size="lg"
                >
                  Cancel
                </NeonButton>
              </div>
            </div>
          </div>
        </NeonCard>
      </div>
    );
  }

  // ── Generating animation ──
  return (
    <div className="space-y-4">
      <NeonCard variant="cyan" glow>
        <div className="space-y-5 py-2">
          <div className="text-center space-y-3">
            <div className="relative inline-block">
              <div className="w-20 h-20 mx-auto rounded-full border-2 border-neon-cyan/20 relative">
                <div
                  className="absolute inset-0 rounded-full border-2 border-transparent border-t-neon-cyan"
                  style={{ animation: "spin 1s linear infinite" }}
                />
                <div
                  className="absolute inset-0 rounded-full border-2 border-transparent border-b-neon-green/40"
                  style={{ animation: "spin 2s linear infinite reverse" }}
                />
                <div className="absolute inset-0 flex items-center justify-center">
                  <span
                    className="text-neon-cyan text-2xl"
                    style={{
                      filter: "drop-shadow(0 0 12px rgba(0,255,255,0.8))",
                      animation: "pulse 2s ease-in-out infinite",
                    }}
                  >
                    &#9670;
                  </span>
                </div>
              </div>
            </div>

            <div className="text-xs font-bold uppercase tracking-[0.25em] text-neon-cyan/80">
              {phase >= 0 && phase < PHASES.length ? PHASES[phase] : "PREPARING"}
              <span className="animate-pulse">...</span>
            </div>

            <div className="flex justify-center gap-2">
              {PHASES.slice(0, -1).map((_, i) => (
                <div
                  key={i}
                  className={`h-1 rounded-full transition-all duration-500 ${
                    i < phase
                      ? "w-8 bg-neon-cyan shadow-[0_0_6px_rgba(0,255,255,0.6)]"
                      : i === phase
                        ? "w-8 bg-neon-cyan/60 animate-pulse"
                        : "w-4 bg-white/10"
                  }`}
                />
              ))}
            </div>
          </div>

          <div className="bg-black/60 rounded-lg border border-neon-cyan/10 p-4 font-mono text-xs overflow-hidden min-h-[140px] flex items-center justify-center">
            {phase >= 0 && phase <= 3 && (
              <div className="w-full space-y-0.5">
                {phase === 0 && entropyLines.length === 0 && (
                  <div className="text-center text-neon-cyan/30 animate-pulse py-8">
                    Initializing secure random number generator...
                  </div>
                )}
                {entropyLines.map((line, i) => (
                  <div
                    key={i}
                    className="text-neon-green/40 tracking-[0.2em] text-center"
                    style={{ opacity: 0.15 + (i / entropyLines.length) * 0.85 }}
                  >
                    {line}
                  </div>
                ))}
              </div>
            )}
            {phase === 4 && (
              <div className="flex flex-col items-center gap-3 w-full">
                <div className="text-[10px] uppercase tracking-[0.3em] text-white/30">
                  Resolving Address
                </div>
                <div className="text-lg tracking-wide break-all text-center px-2 leading-relaxed">
                  {displayAddr.split("").map((ch, i) => (
                    <span
                      key={i}
                      className={i < revealCount ? "text-neon-cyan" : "text-neon-cyan/20"}
                      style={
                        i < revealCount
                          ? { filter: "drop-shadow(0 0 4px rgba(0,255,255,0.6))", transition: "all 0.15s" }
                          : {}
                      }
                    >
                      {ch}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {phase === 5 && (
              <div className="flex flex-col items-center gap-3 w-full">
                <div className="text-[10px] uppercase tracking-[0.3em] text-neon-green/70 font-bold">
                  &#10003; Wallet Ready
                </div>
                <div
                  className="text-lg tracking-wide break-all text-center px-2 text-neon-cyan leading-relaxed"
                  style={{ filter: "drop-shadow(0 0 8px rgba(0,255,255,0.5))" }}
                >
                  {displayAddr}
                </div>
              </div>
            )}
          </div>
        </div>
      </NeonCard>

      {/* Import existing key link — shown during/after generation */}
      {!generating && phase < 0 && (
        <button
          onClick={() => { setImportMode(true); setAutoStarted(true); }}
          className="w-full text-[11px] uppercase tracking-widest text-white/25 hover:text-neon-cyan/60 py-2 border border-dashed border-white/10 hover:border-neon-cyan/20 rounded-lg transition-all"
        >
          Import Existing Key Instead
        </button>
      )}
    </div>
  );
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Step 2: Identity & Personality
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

function IdentityStep({
  showToast,
  onNext,
}: {
  showToast: Props["showToast"];
  onNext: () => void;
}) {
  const [files, setFiles] = useState<BootstrapFile[]>([]);
  const [editingFile, setEditingFile] = useState<string | null>(null);
  const [editContent, setEditContent] = useState("");
  const [saving, setSaving] = useState(false);
  const [created, setCreated] = useState(false);

  const loadFiles = async () => {
    try {
      const f = await window.go.main.AgentSetupService.GetBootstrapFiles();
      setFiles(f);
      if (f.some((x: BootstrapFile) => x.exists)) setCreated(true);
    } catch { /* noop */ }
  };

  useEffect(() => { loadFiles(); }, []);

  const handleCreateDefaults = async () => {
    try {
      await window.go.main.AgentSetupService.CreateDefaultBootstrapFiles();
      showToast("Default identity files created!", "success");
      setCreated(true);
      loadFiles();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  const handleEdit = (file: BootstrapFile) => {
    setEditingFile(file.name);
    setEditContent(file.content);
  };

  const handleSave = async () => {
    if (!editingFile) return;
    setSaving(true);
    try {
      await window.go.main.AgentSetupService.SaveBootstrapFile(editingFile, editContent);
      showToast(`${editingFile} saved!`, "success");
      setEditingFile(null);
      loadFiles();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setSaving(false);
    }
  };

  const hasAnyFiles = files.some((f) => f.exists);

  return (
    <div className="space-y-5">
      {!hasAnyFiles && !created && (
        <NeonCard variant="purple" glow>
          <div className="text-center space-y-4 py-2">
            <div className="w-14 h-14 mx-auto rounded-2xl bg-neon-purple/10 border border-neon-purple/20 flex items-center justify-center">
              <span className="text-2xl text-neon-purple/50">&#9733;</span>
            </div>
            <div className="space-y-1">
              <h3 className="text-sm font-bold uppercase tracking-widest text-white/70">Agent Identity</h3>
              <p className="text-xs text-white/35 max-w-xs mx-auto">
                Create personality, identity, and behavior files for your agent.
              </p>
            </div>
            <NeonButton onClick={handleCreateDefaults} size="lg" className="w-full">
              Create Agent Identity Files
            </NeonButton>
          </div>
        </NeonCard>
      )}

      {editingFile && (
        <NeonCard variant="pink" glow>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold uppercase tracking-widest text-neon-pink">{editingFile}</h3>
              <button onClick={() => setEditingFile(null)} className="text-xs text-white/30 hover:text-white/60">Cancel</button>
            </div>
            <textarea
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              rows={14}
              className="w-full bg-black/40 border-2 border-neon-purple/20 rounded-lg px-4 py-3 text-sm text-white/80 font-mono leading-relaxed focus:outline-none focus:border-neon-pink/40 focus:shadow-neon-pink transition-all resize-y"
            />
            <NeonButton onClick={handleSave} disabled={saving} className="w-full">
              {saving ? "Saving..." : "Save"}
            </NeonButton>
          </div>
        </NeonCard>
      )}

      {!editingFile && hasAnyFiles &&
        files.map((file) => (
          <NeonCard key={file.name} variant="purple">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <h3 className="text-sm font-bold uppercase tracking-widest text-white/80">{file.name}</h3>
                <p className="text-xs text-white/30 mt-0.5">{fileDescriptions[file.name] || "Agent configuration file"}</p>
                {file.exists && (
                  <p className="text-[10px] text-neon-green/40 mt-1 font-mono">{file.content.split("\n").length} lines</p>
                )}
              </div>
              <div className="flex items-center gap-2">
                {file.exists ? (
                  <NeonButton onClick={() => handleEdit(file)} variant="ghost" size="sm">Edit</NeonButton>
                ) : (
                  <NeonButton
                    onClick={() => { setEditingFile(file.name); setEditContent(`# ${file.name.replace(".md", "")}\n\n`); }}
                    size="sm"
                  >
                    Create
                  </NeonButton>
                )}
              </div>
            </div>
          </NeonCard>
        ))}

      {hasAnyFiles && !editingFile && (
        <NeonButton onClick={onNext} size="lg" className="w-full">
          Next: Install Skills &rarr;
        </NeonButton>
      )}
    </div>
  );
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Step 3: Skills
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

function SkillsStep({
  showToast,
  onComplete,
}: {
  showToast: Props["showToast"];
  onComplete: () => void;
}) {
  const [skills, setSkills] = useState<SkillEntry[]>([]);
  const [installRepo, setInstallRepo] = useState("");
  const [installing, setInstalling] = useState(false);
  const [builtinsInstalled, setBuiltinsInstalled] = useState(false);

  const loadSkills = async () => {
    try {
      const s = await window.go.main.AgentSetupService.ListSkills();
      setSkills(s || []);
      if (s && s.length > 0) setBuiltinsInstalled(true);
    } catch { /* noop */ }
  };

  useEffect(() => { loadSkills(); }, []);

  const handleInstallBuiltins = async () => {
    setInstalling(true);
    try {
      await window.go.main.AgentSetupService.InstallBuiltinSkills();
      showToast("Builtin skills installed!", "success");
      setBuiltinsInstalled(true);
      loadSkills();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setInstalling(false);
    }
  };

  const handleInstallSkill = async () => {
    if (!installRepo.trim()) return;
    setInstalling(true);
    try {
      await window.go.main.AgentSetupService.InstallSkill(installRepo.trim());
      showToast("Skill installed!", "success");
      setInstallRepo("");
      loadSkills();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setInstalling(false);
    }
  };

  const handleRemoveSkill = async (name: string) => {
    try {
      await window.go.main.AgentSetupService.RemoveSkill(name);
      showToast(`${name} removed`, "success");
      loadSkills();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  return (
    <div className="space-y-5">
      {!builtinsInstalled && skills.length === 0 && (
        <NeonCard variant="cyan" glow>
          <div className="text-center space-y-4 py-2">
            <div className="w-14 h-14 mx-auto rounded-2xl bg-neon-cyan/10 border border-neon-cyan/20 flex items-center justify-center">
              <span className="text-2xl text-neon-cyan/50">&#9889;</span>
            </div>
            <div className="space-y-1">
              <h3 className="text-sm font-bold uppercase tracking-widest text-white/70">Agent Skills</h3>
              <p className="text-xs text-white/35 max-w-xs mx-auto">Install builtin skills to give your agent superpowers.</p>
            </div>
            <NeonButton onClick={handleInstallBuiltins} disabled={installing} size="lg" className="w-full">
              {installing ? "Installing..." : "Install Builtin Skills"}
            </NeonButton>
          </div>
        </NeonCard>
      )}

      {(builtinsInstalled || skills.length > 0) && (
        <>
          <NeonCard variant="cyan">
            <div className="space-y-3">
              <h3 className="text-sm font-bold uppercase tracking-widest text-white/80">Install Custom Skill</h3>
              <div className="flex gap-2">
                <div className="flex-1">
                  <NeonInput value={installRepo} onChange={setInstallRepo} placeholder="owner/repo/skill-name (GitHub)" />
                </div>
                <NeonButton onClick={handleInstallSkill} disabled={!installRepo.trim() || installing} size="sm">
                  {installing ? "..." : "Install"}
                </NeonButton>
              </div>
            </div>
          </NeonCard>

          {skills.map((skill) => (
            <NeonCard key={skill.name} variant="cyan">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h4 className="text-sm font-bold text-white/80">{skill.name}</h4>
                    <span className="text-[10px] text-neon-cyan/60 bg-neon-cyan/10 px-2 py-0.5 rounded border border-neon-cyan/20 uppercase tracking-widest font-bold">
                      {skill.source}
                    </span>
                  </div>
                  {skill.description && <p className="text-xs text-white/30 mt-1">{skill.description}</p>}
                </div>
                <NeonButton onClick={() => handleRemoveSkill(skill.name)} variant="danger" size="sm">Remove</NeonButton>
              </div>
            </NeonCard>
          ))}
        </>
      )}

      {(builtinsInstalled || skills.length > 0) && (
        <NeonButton onClick={onComplete} size="lg" className="w-full" variant="green">
          Complete Setup &rarr; Dashboard
        </NeonButton>
      )}
    </div>
  );
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Main Wizard
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

export default function AgentSetupPage({ showToast, onComplete }: Props) {
  const [step, setStep] = useState(0);

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="text-center space-y-2">
        <h2 className="text-3xl font-bold uppercase tracking-[0.2em] text-glow-pink">
          {STEPS[step].label}
        </h2>
        <p className="text-white/35 text-sm">
          {step === 0 && "Generate your agent's on-chain wallet"}
          {step === 1 && "Define your agent's personality and identity"}
          {step === 2 && "Install skills to extend your agent's abilities"}
        </p>
      </div>

      {/* Progress bar */}
      <div className="flex items-center gap-2 px-4">
        {STEPS.map((s, i) => (
          <div key={s.key} className="flex items-center gap-2 flex-1">
            <div
              className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold transition-all duration-500 ${
                i < step
                  ? "bg-neon-green/20 text-neon-green border border-neon-green/40 shadow-[0_0_10px_rgba(0,255,65,0.3)]"
                  : i === step
                    ? "bg-neon-pink/20 text-neon-pink border border-neon-pink/40 shadow-[0_0_10px_rgba(255,0,146,0.3)] animate-pulse"
                    : "bg-white/5 text-white/20 border border-white/10"
              }`}
            >
              {i < step ? "✓" : s.num}
            </div>
            <span
              className={`text-[10px] uppercase tracking-widest font-bold hidden sm:block ${
                i < step ? "text-neon-green/60" : i === step ? "text-white/60" : "text-white/20"
              }`}
            >
              {s.label}
            </span>
            {i < STEPS.length - 1 && (
              <div className={`flex-1 h-px transition-all duration-500 ${i < step ? "bg-neon-green/30" : "bg-white/10"}`} />
            )}
          </div>
        ))}
      </div>

      {step === 0 && <WalletStep showToast={showToast} onNext={() => setStep(1)} />}
      {step === 1 && <IdentityStep showToast={showToast} onNext={() => setStep(2)} />}
      {step === 2 && <SkillsStep showToast={showToast} onComplete={onComplete || (() => {})} />}
    </div>
  );
}
