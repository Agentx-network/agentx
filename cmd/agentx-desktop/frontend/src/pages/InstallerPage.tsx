import { useState, useEffect } from "react";
import type { PlatformInfo, DownloadProgress } from "../lib/types";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import DownloadProgressBar from "../components/DownloadProgress";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
  onComplete?: () => void;
}

export default function InstallerPage({ showToast, onComplete }: Props) {
  const [platform, setPlatform] = useState<PlatformInfo | null>(null);
  const [latestVersion, setLatestVersion] = useState("");
  const [installing, setInstalling] = useState(false);
  const [progress, setProgress] = useState<DownloadProgress | null>(null);
  const [installed, setInstalled] = useState(false);

  useEffect(() => {
    const load = async () => {
      try {
        const p = await window.go.main.InstallerService.DetectPlatform();
        setPlatform(p);
        setInstalled(p.binaryExists);
        const ver = await window.go.main.InstallerService.GetLatestRelease();
        setLatestVersion(ver);
      } catch { /* noop */ }
    };
    load();

    window.runtime.EventsOn("download:progress", (data: DownloadProgress) => {
      setProgress(data);
    });
    return () => window.runtime.EventsOff("download:progress");
  }, []);

  const install = async () => {
    setInstalling(true);
    setProgress(null);
    try {
      await window.go.main.InstallerService.InstallBinary();
      setInstalled(true);
      showToast("AgentX installed successfully!", "success");
      const p = await window.go.main.InstallerService.DetectPlatform();
      setPlatform(p);
    } catch (e: any) {
      showToast(`Install failed: ${e}`, "error");
    } finally {
      setInstalling(false);
    }
  };

  const installService = async () => {
    try {
      await window.go.main.InstallerService.InstallService();
      showToast("Service installed and started", "success");
    } catch (e: any) {
      showToast(`Service install failed: ${e}`, "error");
    }
  };

  const hasUpdate =
    installed &&
    latestVersion &&
    platform?.version &&
    !platform.version.includes(latestVersion.replace("v", ""));

  return (
    <div className="max-w-xl mx-auto space-y-6">
      <div className="text-center space-y-2">
        <h2 className="text-3xl font-bold">Install AgentX</h2>
        <p className="text-white/40 text-sm">
          Download and install the AgentX binary to get started.
        </p>
      </div>

      <NeonCard title="Platform">
        {platform ? (
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-white/50">OS</span>
              <span className="text-white capitalize">{platform.os}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-white/50">Architecture</span>
              <span className="text-white">{platform.arch}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-white/50">Install Directory</span>
              <span className="text-white font-mono text-xs">{platform.installDir}</span>
            </div>
            {platform.binaryExists && (
              <div className="flex justify-between">
                <span className="text-white/50">Installed Version</span>
                <span className="text-neon-cyan">{platform.version || "unknown"}</span>
              </div>
            )}
            {latestVersion && (
              <div className="flex justify-between">
                <span className="text-white/50">Latest Release</span>
                <span className={hasUpdate ? "text-neon-pink font-medium" : "text-neon-pink"}>
                  {latestVersion}{hasUpdate ? " — update available!" : ""}
                </span>
              </div>
            )}
          </div>
        ) : (
          <p className="text-white/40 text-sm">Detecting platform...</p>
        )}
      </NeonCard>

      {!installed ? (
        <NeonCard>
          <div className="space-y-4">
            <p className="text-sm text-white/60">
              Download the AgentX binary from GitHub releases.
            </p>
            {progress && <DownloadProgressBar progress={progress} />}
            <NeonButton onClick={install} disabled={installing} size="lg" className="w-full">
              {installing ? "Installing..." : "Install AgentX"}
            </NeonButton>
          </div>
        </NeonCard>
      ) : (
        <NeonCard>
          <div className="space-y-5">
            <div className="flex items-center gap-3 text-green-400">
              <div className="w-8 h-8 rounded-full bg-green-500/20 flex items-center justify-center text-lg">
                ✓
              </div>
              <div>
                <p className="text-sm font-medium">AgentX binary installed</p>
                <p className="text-xs text-white/40">{platform?.binaryPath}</p>
              </div>
            </div>

            {hasUpdate && (
              <div className="border-t border-white/10 pt-4 space-y-3">
                <p className="text-sm text-white/60">
                  A newer version ({latestVersion}) is available. You have {platform?.version}.
                </p>
                {progress && <DownloadProgressBar progress={progress} />}
                <NeonButton onClick={install} disabled={installing} size="lg" className="w-full">
                  {installing ? "Updating..." : `Update to ${latestVersion}`}
                </NeonButton>
              </div>
            )}

            <div className="border-t border-white/10 pt-4 space-y-3">
              <p className="text-xs text-white/40">
                Optional: install as a system service so the gateway starts automatically on boot.
              </p>
              <NeonButton onClick={installService} variant="ghost" size="sm">
                Install as System Service
              </NeonButton>
            </div>

            {onComplete && (
              <NeonButton onClick={onComplete} size="lg" className="w-full">
                Continue to Setup →
              </NeonButton>
            )}
          </div>
        </NeonCard>
      )}
    </div>
  );
}
