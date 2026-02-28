import { useState, useEffect } from "react";
import type { GatewayStatus } from "../lib/types";
import NeonButton from "../components/ui/NeonButton";
import HealthCard from "../components/HealthCard";
import ChannelsCard from "../components/ChannelsCard";
import ModelsCard from "../components/ModelsCard";
import LogViewer from "../components/LogViewer";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

export default function DashboardPage({ showToast }: Props) {
  const [status, setStatus] = useState<GatewayStatus | null>(null);

  const fetchStatus = async () => {
    try {
      const s = await window.go.main.DashboardService.GetStatus();
      setStatus(s);
    } catch { /* noop */ }
  };

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleStart = async () => {
    try {
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Dashboard</h2>
        <div className="flex gap-2">
          {!status?.running ? (
            <NeonButton onClick={handleStart} size="sm">Start</NeonButton>
          ) : (
            <>
              <NeonButton onClick={handleRestart} variant="ghost" size="sm">Restart</NeonButton>
              <NeonButton onClick={handleStop} variant="danger" size="sm">Stop</NeonButton>
            </>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <HealthCard status={status} />
        <ChannelsCard channels={status?.channels ?? []} />
      </div>

      <ModelsCard models={status?.models ?? []} />
      <LogViewer />
    </div>
  );
}
