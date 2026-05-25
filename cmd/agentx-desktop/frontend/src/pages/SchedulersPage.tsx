import { useCallback, useEffect, useState } from "react";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import type { SchedulerInfo } from "../lib/types";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

// humanInterval turns a millisecond duration into a short human phrase.
function humanInterval(ms: number): string {
  const sec = Math.round(ms / 1000);
  if (sec < 60) return `${sec}s`;
  const min = Math.round(sec / 60);
  if (min < 60) return `${min} min`;
  const hr = Math.round(min / 60);
  if (hr < 24) return `${hr} hr`;
  const d = Math.round(hr / 24);
  return `${d} day${d === 1 ? "" : "s"}`;
}

// fireDescription summarises when a scheduled job will trigger, in plain English.
function fireDescription(s: SchedulerInfo): string {
  switch (s.kind) {
    case "at": {
      if (!s.atMs) return "one-time";
      const ms = s.atMs - Date.now();
      if (ms <= 0) return "due now";
      return `in ${humanInterval(ms)}`;
    }
    case "every":
      return s.everyMs ? `every ${humanInterval(s.everyMs)}` : "recurring";
    case "cron":
      return s.cronExpr ? `cron: ${s.cronExpr}` : "cron";
    default:
      return s.kind || "—";
  }
}

// runawayHint flags suspicious schedulers — a recurring shell command at a
// short interval is the exact runaway pattern we want users to spot.
function runawayHint(s: SchedulerInfo): string | null {
  if (s.command && s.kind === "every" && s.everyMs && s.everyMs < 60_000) {
    return "Suspicious: shell command fires < 1 minute. Likely a runaway.";
  }
  return null;
}

// destinationLabel turns the (channel, chatID) pair into a user-readable target.
function destinationLabel(s: SchedulerInfo): string {
  const ch = (s.channel || "").toLowerCase();
  if (!ch || ch === "desktop") return "this app";
  if (ch === "cli") return "terminal";
  return ch.charAt(0).toUpperCase() + ch.slice(1);
}

export default function SchedulersPage({ showToast }: Props) {
  const [items, setItems] = useState<SchedulerInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [confirmAll, setConfirmAll] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const list = await window.go.main.SchedulersService.ListSchedulers();
      setItems(list ?? []);
    } catch (e: any) {
      showToast(`Couldn't load schedulers: ${e}`, "error");
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [showToast]);

  useEffect(() => {
    load();
    // Refresh every 8s so the list reflects newly-created or fired jobs.
    const t = setInterval(load, 8000);
    return () => clearInterval(t);
  }, [load]);

  const removeOne = async (id: string) => {
    setBusy(true);
    try {
      const n = await window.go.main.SchedulersService.RemoveScheduler(id);
      if (n > 0) {
        showToast("Scheduler removed.", "success");
        setItems((prev) => prev.filter((s) => s.id !== id));
      } else {
        showToast("Scheduler was already gone.", "error");
        load();
      }
    } catch (e: any) {
      showToast(`Couldn't remove: ${e}`, "error");
    } finally {
      setBusy(false);
    }
  };

  const removeAll = async () => {
    setBusy(true);
    try {
      const n = await window.go.main.SchedulersService.RemoveAllSchedulers();
      showToast(n === 0 ? "Nothing to cancel." : `Cancelled ${n} scheduler${n === 1 ? "" : "s"}. ✅`, "success");
      setItems([]);
      setConfirmAll(false);
    } catch (e: any) {
      showToast(`Couldn't cancel all: ${e}`, "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex items-end justify-between">
        <div>
          <h1 className="text-3xl font-bold uppercase tracking-[0.2em] text-glow-pink">Schedulers</h1>
          <p className="text-white/40 text-sm mt-2">
            Every reminder and scheduled task currently active. Remove anything that's misbehaving — the change applies immediately.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <NeonButton variant="ghost" size="sm" onClick={load} disabled={loading || busy}>
            {loading ? "Refreshing…" : "↻ Refresh"}
          </NeonButton>
          {items.length > 0 && !confirmAll && (
            <NeonButton variant="danger" size="sm" onClick={() => setConfirmAll(true)} disabled={busy}>
              Remove all
            </NeonButton>
          )}
          {confirmAll && (
            <>
              <NeonButton variant="ghost" size="sm" onClick={() => setConfirmAll(false)} disabled={busy}>
                Cancel
              </NeonButton>
              <NeonButton variant="danger" size="sm" onClick={removeAll} disabled={busy}>
                {busy ? "Removing…" : `Yes, remove all ${items.length}`}
              </NeonButton>
            </>
          )}
        </div>
      </div>

      {loading && items.length === 0 ? (
        <NeonCard>
          <p className="text-center text-white/35 py-8 text-sm">Loading schedulers…</p>
        </NeonCard>
      ) : items.length === 0 ? (
        <NeonCard>
          <div className="text-center py-12">
            <div className="text-5xl mb-4 opacity-30">⏰</div>
            <p className="text-white/60 text-sm">No active schedulers.</p>
            <p className="text-white/30 text-xs mt-1">
              Ask the agent in chat to "remind me in 5 minutes to …" and it'll appear here.
            </p>
          </div>
        </NeonCard>
      ) : (
        <div className="space-y-3">
          {items.map((s) => {
            const warn = runawayHint(s);
            return (
              <NeonCard key={s.id} variant={warn ? "purple" : undefined}>
                <div className="flex items-start gap-4">
                  <div className="text-2xl flex-shrink-0 mt-0.5">{s.command ? "⚙️" : "⏰"}</div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="text-sm font-bold text-white/90 truncate">{s.name || s.message || s.id}</h3>
                      {s.command && (
                        <span className="text-[10px] uppercase tracking-widest font-bold bg-neon-purple/15 text-neon-purple px-2 py-0.5 rounded border border-neon-purple/30">
                          Command
                        </span>
                      )}
                      {!s.enabled && (
                        <span className="text-[10px] uppercase tracking-widest font-bold bg-white/10 text-white/40 px-2 py-0.5 rounded">
                          Disabled
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-white/55 mt-1 truncate">
                      <span className="text-neon-cyan">{fireDescription(s)}</span>
                      <span className="text-white/30 mx-2">·</span>
                      <span>to {destinationLabel(s)}</span>
                    </p>
                    {s.command && (
                      <p className="text-[11px] text-white/40 mt-1 font-mono truncate">
                        $ {s.command}
                      </p>
                    )}
                    {!s.command && s.message && (
                      <p className="text-xs text-white/50 mt-1 truncate">"{s.message}"</p>
                    )}
                    {warn && (
                      <p className="text-xs text-red-400 mt-2 flex items-center gap-1.5">
                        <span>⚠</span>
                        <span>{warn}</span>
                      </p>
                    )}
                  </div>
                  <NeonButton variant="danger" size="sm" onClick={() => removeOne(s.id)} disabled={busy}>
                    Remove
                  </NeonButton>
                </div>
              </NeonCard>
            );
          })}
        </div>
      )}

      <p className="text-center text-[11px] text-white/25 uppercase tracking-widest pt-2">
        {items.length === 0
          ? ""
          : `${items.length} scheduler${items.length === 1 ? "" : "s"} active`}
      </p>
    </div>
  );
}
