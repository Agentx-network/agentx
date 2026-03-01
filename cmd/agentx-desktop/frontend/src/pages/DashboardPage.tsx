import { useState, useEffect, useRef } from "react";
import type { GatewayStatus } from "../lib/types";
import NeonButton from "../components/ui/NeonButton";

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

export default function DashboardPage({ showToast }: Props) {
  const [status, setStatus] = useState<GatewayStatus | null>(null);
  const [logs, setLogs] = useState("");
  const [logsLoading, setLogsLoading] = useState(false);
  const [files, setFiles] = useState<BootstrapFile[]>([]);
  const [skills, setSkills] = useState<SkillEntry[]>([]);
  const [showLogs, setShowLogs] = useState(false);
  const logRef = useRef<HTMLPreElement>(null);

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

  useEffect(() => {
    fetchStatus();
    fetchLogs();
    loadAgent();
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

  return (
    <div className="space-y-5 overflow-y-auto">
      {/* Header */}
      <div className="flex items-center justify-between">
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

      {/* Hero status card */}
      <div className={`relative overflow-hidden rounded-xl border p-5 ${
        running
          ? "bg-gradient-to-br from-neon-green/[0.04] to-neon-cyan/[0.02] border-neon-green/20"
          : "bg-gradient-to-br from-red-500/[0.04] to-white/[0.01] border-red-500/20"
      }`}>
        <div className="flex items-center gap-4">
          <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${
            running
              ? "bg-neon-green/10 shadow-[0_0_20px_rgba(0,255,65,0.15)]"
              : "bg-red-500/10"
          }`}>
            <div className={`w-4 h-4 rounded-full ${
              running
                ? "bg-neon-green shadow-[0_0_12px_rgba(0,255,65,0.6)] animate-pulse"
                : "bg-red-500/60"
            }`} />
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
              <div className="text-[10px] uppercase tracking-widest text-white/25">Model</div>
              <div className="text-sm text-white/70 font-medium">{activeModel.modelName}</div>
              <div className="text-[10px] text-white/20 font-mono">{activeModel.model}</div>
            </div>
          )}
        </div>
      </div>

      {/* Info grid */}
      <div className="grid grid-cols-3 gap-3">
        {/* Channels */}
        <div className="glass-card p-4">
          <div className="text-[10px] uppercase tracking-widest text-white/25 mb-2">Channels</div>
          {activeChannels.length > 0 ? (
            <div className="space-y-1.5">
              {activeChannels.map(ch => (
                <div key={ch.name} className="flex items-center gap-2">
                  <div className="w-1.5 h-1.5 rounded-full bg-neon-cyan shadow-[0_0_6px_rgba(0,255,255,0.4)]" />
                  <span className="text-xs text-white/60">{ch.name}</span>
                </div>
              ))}
            </div>
          ) : (
            <span className="text-xs text-white/15">None active</span>
          )}
        </div>

        {/* Identity files */}
        <div className="glass-card p-4">
          <div className="text-[10px] uppercase tracking-widest text-white/25 mb-2">Identity</div>
          {identityFiles.length > 0 ? (
            <div className="space-y-1.5">
              {identityFiles.map(f => (
                <div key={f.name} className="flex items-center gap-2">
                  <div className="w-1.5 h-1.5 rounded-full bg-neon-purple shadow-[0_0_6px_rgba(168,85,247,0.4)]" />
                  <span className="text-xs text-white/60">{f.name.replace(".md", "")}</span>
                </div>
              ))}
            </div>
          ) : (
            <span className="text-xs text-white/15">Not configured</span>
          )}
        </div>

        {/* Skills */}
        <div className="glass-card p-4">
          <div className="text-[10px] uppercase tracking-widest text-white/25 mb-2">
            Skills <span className="text-neon-cyan/40">({skills.length})</span>
          </div>
          {skills.length > 0 ? (
            <div className="space-y-1.5">
              {skills.slice(0, 5).map(s => (
                <div key={s.name} className="flex items-center gap-2">
                  <div className="w-1.5 h-1.5 rounded-full bg-neon-cyan shadow-[0_0_6px_rgba(0,255,255,0.3)]" />
                  <span className="text-xs text-white/60 truncate">{s.name}</span>
                </div>
              ))}
              {skills.length > 5 && (
                <span className="text-[10px] text-white/20">+{skills.length - 5} more</span>
              )}
            </div>
          ) : (
            <span className="text-xs text-white/15">None installed</span>
          )}
        </div>
      </div>

      {/* Models configured */}
      {configuredModels.length > 1 && (
        <div className="glass-card p-4">
          <div className="text-[10px] uppercase tracking-widest text-white/25 mb-2">
            Models <span className="text-neon-green/40">({configuredModels.length})</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {configuredModels.map(m => (
              <span key={m.modelName} className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-neon-green/5 border border-neon-green/10 text-xs text-white/50">
                <div className="w-1.5 h-1.5 rounded-full bg-neon-green shadow-[0_0_4px_rgba(0,255,65,0.4)]" />
                {m.modelName}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Logs - collapsible */}
      <div className="glass-card overflow-hidden">
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
                className="text-[10px] text-neon-cyan/50 hover:text-neon-cyan uppercase tracking-widest font-bold transition-colors"
              >
                {logsLoading ? "..." : "Refresh"}
              </button>
            )}
            <span className={`text-white/20 text-xs transition-transform ${showLogs ? "rotate-180" : ""}`}>
              ▼
            </span>
          </div>
        </button>
        {showLogs && (
          <pre
            ref={logRef}
            className="px-4 py-3 text-xs text-neon-green/60 font-mono whitespace-pre-wrap leading-relaxed overflow-auto max-h-64 border-t border-white/[0.04]"
          >
            {logs || "No logs available"}
          </pre>
        )}
      </div>
    </div>
  );
}
