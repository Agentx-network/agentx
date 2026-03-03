import { useState, useEffect, useRef } from "react";
import type { GatewayStatus } from "../lib/types";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import NeonInput from "../components/ui/NeonInput";
import GlowCard from "../components/ui/GlowCard";
import { useStaggeredEntrance } from "../hooks/useStaggeredEntrance";
import { useCountUp } from "../hooks/useCountUp";
import agentHero from "../assets/agent-hero.gif";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

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

interface TokenBalance {
  symbol: string;
  name: string;
  contract: string;
  balance: string;
  decimals: number;
}

interface RegistryInfo {
  registered: boolean;
  agentName: string;
  agentId: string;
  address: string;
  chain: string;
  metadata: string;
  txHash: string;
  timestamp: string;
}

const TOKEN_ICONS: Record<string, string> = {
  BNB: "🟡",
  USDT: "🟢",
  USDC: "🔵",
  BUSD: "🟡",
  DAI: "🟠",
};

function PulseRing({ color }: { color: string }) {
  return (
    <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
      <div
        className="absolute w-3.5 h-3.5 rounded-full"
        style={{
          border: `1.5px solid ${color}`,
          animation: "pulse-ring 2s ease-out infinite",
        }}
      />
      <div
        className="absolute w-3.5 h-3.5 rounded-full"
        style={{
          border: `1.5px solid ${color}`,
          animation: "pulse-ring 2s ease-out infinite 0.5s",
        }}
      />
    </div>
  );
}

export default function DashboardPage({ showToast }: Props) {
  const [status, setStatus] = useState<GatewayStatus | null>(null);
  const [logs, setLogs] = useState("");
  const [logsLoading, setLogsLoading] = useState(false);
  const [files, setFiles] = useState<BootstrapFile[]>([]);
  const [skills, setSkills] = useState<SkillEntry[]>([]);
  const [showLogs, setShowLogs] = useState(true);
  const logRef = useRef<HTMLPreElement>(null);
  const [walletAddr, setWalletAddr] = useState("");
  const [balances, setBalances] = useState<TokenBalance[]>([]);
  const [registry, setRegistry] = useState<RegistryInfo | null>(null);
  const [registering, setRegistering] = useState(false);
  const [regName, setRegName] = useState("");
  const [fundModal, setFundModal] = useState<{ error: string } | null>(null);
  const [copied, setCopied] = useState(false);

  // Staggered entrance for the 6 major dashboard sections
  const sectionCount = 6;
  const visibleSections = useStaggeredEntrance(sectionCount, 100);

  const fetchStatus = async () => {
    try {
      const s = await window.go.main.DashboardService.GetStatus();
      setStatus(s);
    } catch { /* noop */ }
  };

  const fetchLogs = async () => {
    setLogsLoading(true);
    try {
      const text = await window.go.main.DashboardService.GetLogs(100);
      setLogs(text);
      if (logRef.current) {
        logRef.current.scrollTop = logRef.current.scrollHeight;
      }
    } catch {
      setLogs("No logs available");
    } finally {
      setLogsLoading(false);
    }
  };

  const loadAgent = async () => {
    try {
      const [f, s] = await Promise.all([
        window.go.main.AgentSetupService.GetBootstrapFiles(),
        window.go.main.AgentSetupService.ListSkills(),
      ]);
      setFiles(f);
      setSkills(s || []);
    } catch { /* noop */ }
  };

  const loadWallet = async () => {
    try {
      const w = await window.go.main.WalletService.GetWallet();
      if (w?.address) {
        setWalletAddr(w.address);
        try {
          const b = await window.go.main.WalletService.GetAllBalances();
          setBalances(b || []);
        } catch { /* noop */ }
      }
    } catch { /* no wallet */ }
  };

  const loadRegistry = async () => {
    try {
      const r = await window.go.main.RegistryService.GetRegistration();
      setRegistry(r);
    } catch { /* noop */ }
  };

  useEffect(() => {
    fetchStatus();
    fetchLogs();
    loadAgent();
    loadWallet();
    loadRegistry();
    const statusInterval = setInterval(fetchStatus, 5000);
    const logInterval = setInterval(fetchLogs, 15000);
    return () => { clearInterval(statusInterval); clearInterval(logInterval); };
  }, []);

  const handleStart = async () => {
    try {
      const p = await window.go.main.InstallerService.DetectPlatform();
      if (!p.binaryExists) {
        showToast("AgentX binary not found. Please reinstall from the Install page.", "error");
        return;
      }
      await window.go.main.DashboardService.StartGateway();
      showToast("Gateway starting...", "success");
      setTimeout(fetchStatus, 2000);
    } catch (e: any) {
      showToast(`Start failed: ${e}`, "error");
    }
  };

  const handleStop = async () => {
    try {
      await window.go.main.DashboardService.StopGateway();
      showToast("Gateway stopped", "success");
      setTimeout(fetchStatus, 1000);
    } catch (e: any) {
      showToast(`Stop failed: ${e}`, "error");
    }
  };

  const handleRestart = async () => {
    try {
      await window.go.main.DashboardService.RestartGateway();
      showToast("Gateway restarting...", "success");
      setTimeout(fetchStatus, 2000);
    } catch (e: any) {
      showToast(`Restart failed: ${e}`, "error");
    }
  };

  const handleRegister = async () => {
    const name = regName.trim() || agentName;
    setRegistering(true);
    try {
      const identityContents: Record<string, string> = {};
      for (const f of identityFiles) {
        identityContents[f.name] = f.content;
      }
      const metadata = JSON.stringify({
        skills: skills.map(s => s.name),
        channels: activeChannels.map(c => c.name),
        identity: identityContents,
      });
      const r = await window.go.main.RegistryService.RegisterAgent(name, metadata);
      setRegistry(r);
      showToast("Agent registered in ERC-8004 registry!", "success");
    } catch (e: any) {
      setFundModal({ error: String(e).replace(/^Error:\s*/i, "") });
    } finally {
      setRegistering(false);
    }
  };

  const running = status?.running ?? false;
  const activeChannels = status?.channels?.filter(c => c.enabled) ?? [];
  const configuredModels = status?.models?.filter(m => m.hasKey) ?? [];
  const activeModel = configuredModels[0];
  const identityFiles = files.filter(f => f.exists);

  // Extract agent name from IDENTITY.md if available
  const identityFile = files.find(f => f.name === "IDENTITY.md" && f.exists);
  const agentName = identityFile
    ? (identityFile.content.match(/^#\s*(.+)/m)?.[1] || "AgentX")
    : "AgentX";

  // Count-up animations for stats
  const channelCount = useCountUp(activeChannels.length);
  const skillCount = useCountUp(skills.length);
  const modelCount = useCountUp(configuredModels.length);

  // Helper for staggered section style
  const sectionStyle = (index: number) => ({
    opacity: visibleSections > index ? 1 : 0,
    transform: visibleSections > index ? "translateY(0) scale(1)" : "translateY(20px) scale(0.97)",
    transition: "opacity 0.5s cubic-bezier(0.16, 1, 0.3, 1), transform 0.5s cubic-bezier(0.16, 1, 0.3, 1)",
  });

  // Split logs into lines for staggered animation
  const logLines = logs ? logs.split("\n").slice(-50) : [];

  return (
    <div className="space-y-5 overflow-y-auto">
      {/* Header */}
      <div className="flex items-center justify-between" style={sectionStyle(0)}>
        <h2 className="text-2xl font-bold uppercase tracking-[0.2em] text-glow-pink">Dashboard</h2>
        <div className="flex gap-2">
          {!running ? (
            <NeonButton onClick={handleStart} size="sm">Start</NeonButton>
          ) : (
            <>
              <NeonButton onClick={handleRestart} variant="ghost" size="sm">Restart</NeonButton>
              <NeonButton onClick={handleStop} variant="danger" size="sm">Stop</NeonButton>
            </>
          )}
        </div>
      </div>

      {/* ERC-8004 Agent Registry */}
      <div style={sectionStyle(1)}>
        <NeonCard variant={registry?.registered ? "cyan" : "pink"} glow>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-neon-purple/10 border border-neon-purple/20 flex items-center justify-center">
                  <span className="text-lg drop-shadow-[0_0_8px_rgba(174,0,255,0.6)]">&#9671;</span>
                </div>
                <div>
                  <div className="text-xs uppercase tracking-widest text-neon-purple/80 font-bold">ERC-8004 Agent Registry</div>
                  <div className="text-[10px] text-white/30">On-chain agent identity &amp; capability registration</div>
                </div>
              </div>
              {registry?.registered ? (
                <div className="text-[10px] uppercase tracking-widest text-neon-green/70 bg-neon-green/10 border border-neon-green/20 px-2.5 py-1 rounded-full font-bold">
                  Registered
                </div>
              ) : (
                <div className="text-[10px] uppercase tracking-widest text-yellow-400/70 bg-yellow-400/10 border border-yellow-400/20 px-2.5 py-1 rounded-full font-bold">
                  Not Registered
                </div>
              )}
            </div>

            {registry?.registered ? (
              <div className="space-y-2">
                <div className="grid grid-cols-2 gap-3">
                  <div className="bg-black/30 rounded-lg px-3 py-2 border border-white/[0.04]">
                    <div className="text-[10px] uppercase tracking-widest text-white/30">Agent Name</div>
                    <div className="text-sm text-white/80 font-bold">{registry.agentName}</div>
                  </div>
                  <div className="bg-black/30 rounded-lg px-3 py-2 border border-white/[0.04]">
                    <div className="text-[10px] uppercase tracking-widest text-white/30">Chain</div>
                    <div className="text-sm text-white/80 font-bold">{registry.chain}</div>
                  </div>
                </div>
                <div className="bg-black/30 rounded-lg px-3 py-2 border border-white/[0.04]">
                  <div className="text-[10px] uppercase tracking-widest text-white/30">Wallet Address</div>
                  <div className="text-xs text-neon-cyan/70 font-mono break-all">{registry.address}</div>
                </div>
                {registry.metadata && (
                  <div className="bg-black/30 rounded-lg px-3 py-2 border border-neon-purple/10">
                    <div className="text-[10px] uppercase tracking-widest text-white/30">IPFS Metadata</div>
                    <button
                      onClick={() => window.runtime.BrowserOpenURL(`https://ipfs.agentx.network/ipfs/${registry.metadata.replace("ipfs://", "")}`)}
                      className="text-xs text-neon-purple/70 font-mono break-all hover:text-neon-purple transition-colors cursor-pointer text-left"
                    >
                      {registry.metadata}
                    </button>
                  </div>
                )}
                {registry.txHash && (
                  <div className="bg-black/30 rounded-lg px-3 py-2 border border-neon-green/10">
                    <div className="text-[10px] uppercase tracking-widest text-white/30">Transaction Hash</div>
                    <button
                      onClick={() => window.runtime.BrowserOpenURL(`https://bscscan.com/tx/${registry.txHash}`)}
                      className="text-xs text-neon-green/70 font-mono break-all hover:text-neon-green transition-colors cursor-pointer text-left"
                    >
                      {registry.txHash}
                    </button>
                  </div>
                )}
                <div className="flex items-center justify-between text-[10px] text-white/25 px-1">
                  <span>Registered {new Date(registry.timestamp).toLocaleString()}</span>
                  <span className="font-mono">ERC-8004 · IPFS · BSC</span>
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                {!walletAddr ? (
                  <div className="text-xs text-red-400/60 bg-red-400/5 border border-red-400/10 rounded-lg px-3 py-2">
                    Wallet required — generate one from the Wallet page first.
                  </div>
                ) : (
                  <>
                    <div className="text-xs text-white/40">
                      Register your agent on-chain to enable discovery, reputation tracking, and payment capabilities.
                      Metadata is pinned to IPFS and the URI is stored on-chain via ERC-8004.
                    </div>
                    <div className="flex gap-2">
                      <NeonInput
                        value={regName}
                        onChange={setRegName}
                        placeholder={`Agent name (default: ${agentName})`}
                      />
                      <NeonButton onClick={handleRegister} disabled={registering} size="sm">
                        {registering ? "Registering..." : "Register Agent"}
                      </NeonButton>
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        </NeonCard>
      </div>

      {/* Hero status card */}
      <div style={sectionStyle(2)}>
        <GlowCard
          className={`border p-5 ${
            running
              ? "bg-gradient-to-br from-neon-green/[0.04] to-neon-cyan/[0.02] border-neon-green/20"
              : "bg-gradient-to-br from-red-500/[0.04] to-white/[0.01] border-red-500/20"
          }`}
          color={running ? "rgba(0, 255, 65, 0.25)" : "rgba(255, 50, 50, 0.2)"}
        >
          <div className="flex items-center gap-4">
            <div className="relative">
              <img src={agentHero} alt="" className={`w-12 h-12 rounded-xl border ${
                running
                  ? "border-neon-green/30 shadow-[0_0_20px_rgba(0,255,65,0.2)]"
                  : "border-red-500/20 opacity-60 grayscale"
              }`} />
              <div className="relative">
                <div className={`absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full border-2 border-[#0a0a12] ${
                  running
                    ? "bg-neon-green shadow-[0_0_8px_rgba(0,255,65,0.6)]"
                    : "bg-red-500/60"
                }`} />
                {running && <PulseRing color="rgba(0, 255, 65, 0.4)" />}
              </div>
            </div>
            <div className="flex-1">
              <div className="text-lg font-bold text-white">{agentName}</div>
              <div className="text-xs text-white/40">
                {running ? (
                  <span className="text-neon-green/70">
                    Online {status?.health?.uptime ? `· ${status.health.uptime}` : ""}
                  </span>
                ) : (
                  <span className="text-red-400/70">Offline</span>
                )}
              </div>
            </div>
            {activeModel && (
              <div className="text-right">
                <div className="text-[11px] uppercase tracking-widest text-white/45">Model</div>
                <div className="text-sm text-white/80 font-medium">{activeModel.modelName}</div>
                <div className="text-[11px] text-white/35 font-mono">{activeModel.model}</div>
              </div>
            )}
          </div>
        </GlowCard>
      </div>

      {/* Info grid */}
      <div className="grid grid-cols-3 gap-3" style={sectionStyle(3)}>
        {/* Channels */}
        <div className="glass-card p-4">
          <div className="text-[11px] uppercase tracking-widest text-white/45 mb-2 font-medium">
            Channels <span className="text-neon-cyan/60">({channelCount})</span>
          </div>
          {activeChannels.length > 0 ? (
            <div className="space-y-1.5">
              {activeChannels.map(ch => (
                <div key={ch.name} className="flex items-center gap-2">
                  <div className="w-1.5 h-1.5 rounded-full bg-neon-cyan text-neon-cyan animate-dot-breathe" />
                  <span className="text-xs text-white/70">{ch.name}</span>
                </div>
              ))}
            </div>
          ) : (
            <span className="text-xs text-white/35">None active</span>
          )}
        </div>

        {/* Identity files */}
        <div className="glass-card p-4">
          <div className="text-[11px] uppercase tracking-widest text-white/45 mb-2 font-medium">Identity</div>
          {identityFiles.length > 0 ? (
            <div className="space-y-1.5">
              {identityFiles.map(f => (
                <div key={f.name} className="flex items-center gap-2">
                  <div className="w-1.5 h-1.5 rounded-full bg-neon-purple shadow-[0_0_6px_rgba(168,85,247,0.4)]" />
                  <span className="text-xs text-white/70">{f.name.replace(".md", "")}</span>
                </div>
              ))}
            </div>
          ) : (
            <span className="text-xs text-white/35">Not configured</span>
          )}
        </div>

        {/* Skills */}
        <div className="glass-card p-4">
          <div className="text-[11px] uppercase tracking-widest text-white/45 mb-2 font-medium">
            Skills <span className="text-neon-cyan/60">({skillCount})</span>
          </div>
          {skills.length > 0 ? (
            <div className="space-y-1.5">
              {skills.slice(0, 5).map(s => (
                <div key={s.name} className="flex items-center gap-2">
                  <div className="w-1.5 h-1.5 rounded-full bg-neon-cyan shadow-[0_0_6px_rgba(0,255,255,0.3)]" />
                  <span className="text-xs text-white/70 truncate">{s.name}</span>
                </div>
              ))}
              {skills.length > 5 && (
                <span className="text-[11px] text-white/40">+{skills.length - 5} more</span>
              )}
            </div>
          ) : (
            <span className="text-xs text-white/35">None installed</span>
          )}
        </div>
      </div>

      {/* Wallet & Token Balances */}
      {walletAddr && (
        <div className="glass-card p-4 space-y-3" style={sectionStyle(4)}>
          <div className="flex items-center justify-between">
            <div className="text-[11px] uppercase tracking-widest text-white/45 font-medium">Wallet Balances</div>
            <div className="text-[11px] font-mono text-neon-cyan/50 truncate max-w-[200px]" title={walletAddr}>
              {walletAddr.slice(0, 6)}...{walletAddr.slice(-4)} · BSC
            </div>
          </div>
          <div className="grid grid-cols-5 gap-2">
            {balances.map((tok) => (
              <div key={tok.symbol + tok.contract} className="bg-black/20 rounded-lg px-3 py-2.5 border border-white/[0.04]">
                <div className="flex items-center gap-1.5 mb-1">
                  <span className="text-sm">{TOKEN_ICONS[tok.symbol] || "🪙"}</span>
                  <span className="text-[11px] font-bold text-white/70">{tok.symbol}</span>
                </div>
                <div className="text-xs font-mono text-white/50">{tok.balance}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Models configured */}
      {configuredModels.length > 1 && (
        <div className="glass-card p-4" style={sectionStyle(4)}>
          <div className="text-[11px] uppercase tracking-widest text-white/45 mb-2 font-medium">
            Models <span className="text-neon-green/60">({modelCount})</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {configuredModels.map(m => (
              <span key={m.modelName} className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-neon-green/5 border border-neon-green/10 text-xs text-white/70">
                <div className="w-1.5 h-1.5 rounded-full bg-neon-green shadow-[0_0_4px_rgba(0,255,65,0.4)]" />
                {m.modelName}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Logs - collapsible */}
      <div className="glass-card overflow-hidden" style={sectionStyle(5)}>
        <button
          onClick={() => { setShowLogs(!showLogs); if (!showLogs) fetchLogs(); }}
          className="w-full flex items-center justify-between px-4 py-3 hover:bg-white/[0.02] transition-colors"
        >
          <h3 className="text-xs font-bold uppercase tracking-widest text-white/50">Logs</h3>
          <div className="flex items-center gap-3">
            {showLogs && (
              <button
                onClick={(e) => { e.stopPropagation(); fetchLogs(); }}
                disabled={logsLoading}
                className="text-[11px] text-neon-cyan/70 hover:text-neon-cyan uppercase tracking-widest font-bold transition-colors"
              >
                {logsLoading ? "..." : "Refresh"}
              </button>
            )}
            <span className={`text-white/40 text-xs transition-transform duration-200 ${showLogs ? "rotate-180" : ""}`}>
              ▼
            </span>
          </div>
        </button>
        {showLogs && (
          <pre
            ref={logRef}
            className="px-4 py-3 text-xs text-neon-green/70 font-mono whitespace-pre-wrap leading-relaxed overflow-auto max-h-64 border-t border-white/[0.04]"
          >
            {logLines.length > 0
              ? logLines.map((line, i) => (
                  <div
                    key={i}
                    style={{
                      animation: `log-line-in 0.3s cubic-bezier(0.16, 1, 0.3, 1) ${Math.min(i * 30, 1500)}ms both`,
                    }}
                  >
                    {line}
                  </div>
                ))
              : "No logs available"}
          </pre>
        )}
      </div>
      {/* Fund Wallet Modal */}
      {fundModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-[#0e0e1a] border border-red-500/20 rounded-2xl p-6 max-w-md w-full mx-4 shadow-[0_0_40px_rgba(255,0,80,0.15)] animate-card-enter">
            {/* Header */}
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-xl bg-red-500/10 border border-red-500/20 flex items-center justify-center">
                <span className="text-lg">&#9888;</span>
              </div>
              <div>
                <div className="text-sm font-bold text-white">Registration Failed</div>
                <div className="text-[10px] text-white/40 uppercase tracking-widest">Insufficient BNB for gas</div>
              </div>
            </div>

            {/* Error message */}
            <div className="bg-red-500/5 border border-red-500/10 rounded-lg px-3 py-2 mb-4">
              <div className="text-xs text-red-400/80 break-all">{fundModal.error}</div>
            </div>

            {/* Info */}
            <div className="text-xs text-white/50 mb-4">
              Send BNB to your wallet address below to cover gas fees for on-chain registration. A small amount (0.001–0.005 BNB) is enough.
            </div>

            {/* Wallet address - copyable */}
            <div className="mb-3">
              <div className="text-[10px] uppercase tracking-widest text-white/30 mb-1.5">Your BSC Wallet Address</div>
              <button
                onClick={() => {
                  navigator.clipboard.writeText(walletAddr);
                  setCopied(true);
                  setTimeout(() => setCopied(false), 2000);
                }}
                className="w-full bg-black/40 border border-neon-cyan/20 rounded-lg px-3 py-3 text-left hover:border-neon-cyan/40 transition-colors group cursor-pointer"
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs text-neon-cyan/80 font-mono break-all">{walletAddr}</span>
                  <span className="text-[10px] text-white/30 group-hover:text-neon-cyan/60 ml-2 shrink-0 uppercase tracking-widest transition-colors">
                    {copied ? <span className="animate-check-pop inline-block">✓ Copied</span> : "Copy"}
                  </span>
                </div>
              </button>
            </div>

            {/* Current BNB balance */}
            {balances.length > 0 && (
              <div className="bg-black/30 border border-white/[0.04] rounded-lg px-3 py-2 mb-4">
                <div className="text-[10px] uppercase tracking-widest text-white/30 mb-1">Current Balance</div>
                <div className="flex items-center gap-4">
                  {balances.filter(b => b.symbol === "BNB").map(b => (
                    <div key={b.symbol} className="flex items-center gap-1.5">
                      <span className="text-sm">{TOKEN_ICONS[b.symbol] || "🪙"}</span>
                      <span className="text-xs font-bold text-white/70">{b.balance} {b.symbol}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Actions */}
            <div className="flex gap-2">
              <NeonButton
                onClick={() => {
                  setFundModal(null);
                  loadWallet();
                }}
                variant="ghost"
                size="sm"
              >
                Close
              </NeonButton>
              <NeonButton
                onClick={() => {
                  setFundModal(null);
                  loadWallet();
                  setTimeout(() => handleRegister(), 500);
                }}
                size="sm"
              >
                Retry Registration
              </NeonButton>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
