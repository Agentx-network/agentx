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
      const p = await window.go.main.InstallerService.DetectPlatform();
      setPlatform(p);
      setInstalled(true);
      showToast("Gateway installed successfully!", "success");
    } catch (e: any) {
      showToast(`Install failed: ${e}`, "error");
    } finally {
      setInstalling(false);
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
        <h2 className="text-3xl font-bold uppercase tracking-[0.2em] text-glow-pink">Install AgentX</h2>
        <p className="text-white/35 text-sm">
          Download and install the AgentX gateway to get started.
        </p>
      </div>

      {!installed ? (
        <NeonCard glow>
          <div className="space-y-4">
            {platform && (
              <div className="flex items-center justify-between text-xs text-white/40">
                <span className="uppercase tracking-wide">{platform.os} / {platform.arch}</span>
                {latestVersion && <span className="text-neon-pink">{latestVersion}</span>}
              </div>
            )}
            {progress && <DownloadProgressBar progress={progress} />}
            <NeonButton onClick={install} disabled={installing} size="lg" className="w-full">
              {installing ? "Installing..." : "Install AgentX"}
            </NeonButton>
          </div>
        </NeonCard>
      ) : (
        <NeonCard variant="green">
          <div className="space-y-5">
            <div className="flex items-center gap-3 text-neon-green">
              <div className="w-10 h-10 rounded-full bg-neon-green/15 flex items-center justify-center text-xl shadow-glow-green-sm">
                ✓
              </div>
              <div>
                <p className="text-base font-bold uppercase tracking-wide">Gateway Installed</p>
                <p className="text-xs text-white/30">{platform?.version || latestVersion}</p>
              </div>
            </div>

            {hasUpdate && (
              <div className="border-t border-white/5 pt-4 space-y-3">
                <p className="text-sm text-white/50">
                  A newer version ({latestVersion}) is available.
                </p>
                {progress && <DownloadProgressBar progress={progress} />}
                <NeonButton onClick={install} disabled={installing} size="lg" className="w-full">
                  {installing ? "Updating..." : `Update to ${latestVersion}`}
                </NeonButton>
              </div>
            )}

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
