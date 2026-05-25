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

// humanRelativePast turns a past timestamp into "X ago" phrasing.
function humanRelativePast(ms: number): string {
  const diff = Date.now() - ms;
  if (diff < 0) return "just now";
  if (diff < 45_000) return "just now";
  return `${humanInterval(diff)} ago`;
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
      // Newest first — easier to spot what was just created.
      const sorted = (list ?? []).slice().sort((a, b) => b.createdAtMs - a.createdAtMs);
      setItems(sorted);
    } catch (e: any) {
      showToast(`Couldn't load schedulers: ${e}`, "error");
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [showToast]);

  useEffect(() => {
    load();
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
      showToast(n === 0 ? "Nothing to cancel." : `Cancelled ${n} scheduler${n === 1 ? "" : "s"}.`, "success");
      setItems([]);
      setConfirmAll(false);
    } catch (e: any) {
      showToast(`Couldn't cancel all: ${e}`, "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex flex-col space-y-5 max-w-3xl bg-[#0a0a12]/80 -m-6 p-6 rounded-xl">
      {/* Header — matches Chat/Config pattern: title left, status + actions right,
          bottom border separating header from content. */}
      <div className="flex items-center justify-between gap-4 pb-4 border-b border-neon-pink/15">
        <h2 className="text-2xl font-bold uppercase tracking-[0.2em] text-glow-pink">
          Schedulers
        </h2>
        <div className="flex items-center gap-3 shrink-0">
          {items.length > 0 && (
            <span className="text-[10px] uppercase tracking-widest font-bold text-neon-green/80 bg-neon-green/10 border border-neon-green/25 px-2.5 py-1 rounded-full">
              {items.length} Active
            </span>
          )}
          <div className="flex items-center gap-2">
            <NeonButton variant="ghost" size="sm" onClick={load} disabled={loading || busy} className="whitespace-nowrap">
              {loading ? "Refreshing" : "Refresh"}
            </NeonButton>
            {items.length > 0 && !confirmAll && (
              <NeonButton variant="danger" size="sm" onClick={() => setConfirmAll(true)} disabled={busy} className="whitespace-nowrap">
                Remove all
              </NeonButton>
            )}
            {confirmAll && (
              <>
                <NeonButton variant="ghost" size="sm" onClick={() => setConfirmAll(false)} disabled={busy} className="whitespace-nowrap">
                  Cancel
                </NeonButton>
                <NeonButton variant="danger" size="sm" onClick={removeAll} disabled={busy} className="whitespace-nowrap">
                  {busy ? "Removing…" : `Yes, remove all ${items.length}`}
                </NeonButton>
              </>
            )}
          </div>
        </div>
      </div>

      {loading && items.length === 0 ? (
        <NeonCard>
          <p className="text-center text-white/35 py-8 text-sm">Loading schedulers…</p>
        </NeonCard>
      ) : items.length === 0 ? (
        <NeonCard>
          <div className="text-center py-12">
            <p className="text-xs uppercase tracking-[0.3em] text-white/30 mb-3">No Active Schedulers</p>
            <p className="text-white/55 text-sm">
              Nothing scheduled right now.
            </p>
            <p className="text-white/30 text-xs mt-2">
              Ask the agent in chat — "remind me in 5 minutes to …" — and it'll appear here.
            </p>
          </div>
        </NeonCard>
      ) : (
        <div className="space-y-3">
          {items.map((s) => {
            const warn = runawayHint(s);
            const typeLabel = s.command ? "Command" : "Reminder";
            const typeClasses = s.command
              ? "bg-neon-purple/15 text-neon-purple border-neon-purple/30"
              : "bg-neon-cyan/10 text-neon-cyan border-neon-cyan/30";
            return (
              <NeonCard key={s.id} variant={warn ? "purple" : undefined}>
                <div className="flex items-start gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className={`text-[10px] uppercase tracking-[0.2em] font-bold px-2 py-0.5 rounded border ${typeClasses}`}>
                        {typeLabel}
                      </span>
                      <h3 className="text-sm font-bold text-white/90 truncate">{s.name || s.message || s.id}</h3>
                      {!s.enabled && (
                        <span className="text-[10px] uppercase tracking-widest font-bold bg-white/10 text-white/40 px-2 py-0.5 rounded">
                          Disabled
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-white/55 mt-1.5 truncate">
                      <span className="text-neon-cyan">{fireDescription(s)}</span>
                      <span className="text-white/30 mx-2">·</span>
                      <span>to {destinationLabel(s)}</span>
                      {s.createdAtMs > 0 && (
                        <>
                          <span className="text-white/30 mx-2">·</span>
                          <span className="text-white/40">set {humanRelativePast(s.createdAtMs)}</span>
                        </>
                      )}
                    </p>
                    {s.command && (
                      <p className="text-[11px] text-white/40 mt-1 font-mono truncate">
                        {s.command}
                      </p>
                    )}
                    {!s.command && s.message && (
                      <p className="text-xs text-white/50 mt-1 truncate">"{s.message}"</p>
                    )}
                    {warn && (
                      <p className="text-[11px] text-red-400 mt-2 uppercase tracking-widest font-bold">
                        Runaway pattern: {warn}
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
    </div>
  );
}
