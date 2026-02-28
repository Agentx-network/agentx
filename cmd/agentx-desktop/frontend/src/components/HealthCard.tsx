import type { GatewayStatus } from "../lib/types";
import StatusBadge from "./StatusBadge";

interface Props {
  status: GatewayStatus | null;
}

export default function HealthCard({ status }: Props) {
  return (
    <div className="glass-card p-5">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-white/70">Gateway Health</h3>
        <StatusBadge running={status?.running ?? false} />
      </div>
      {status?.health && (
        <div className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-white/50">Status</span>
            <span className="text-white">{status.health.status}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-white/50">Uptime</span>
            <span className="text-white">{status.health.uptime}</span>
          </div>
        </div>
      )}
      {!status?.running && (
        <p className="text-sm text-white/40 mt-2">Gateway is not running</p>
      )}
    </div>
  );
}
