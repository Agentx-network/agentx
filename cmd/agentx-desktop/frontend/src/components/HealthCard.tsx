import type { GatewayStatus } from "../lib/types";
import StatusBadge from "./StatusBadge";

interface Props {
  status: GatewayStatus | null;
}

export default function HealthCard({ status }: Props) {
  return (
    <div className="glass-card card-pink p-5">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-xs font-bold uppercase tracking-widest text-white/70">Gateway Health</h3>
        <StatusBadge running={status?.running ?? false} />
      </div>
      {status?.health && (
        <div className="space-y-2.5 text-sm">
          <div className="flex justify-between">
            <span className="text-white/40 uppercase text-xs tracking-wide">Status</span>
            <span className="text-white font-medium">{status.health.status}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-white/40 uppercase text-xs tracking-wide">Uptime</span>
            <span className="text-neon-cyan font-mono">{status.health.uptime}</span>
          </div>
        </div>
      )}
      {!status?.running && (
        <p className="text-sm text-white/30 mt-2">Gateway is not running</p>
      )}
    </div>
  );
}
