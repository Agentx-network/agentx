import { useState, useEffect } from "react";
import type { PlatformInfo, DownloadProgress } from "../lib/types";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import DownloadProgressBar from "../components/DownloadProgress";
import logoImg from "../assets/logo.png";

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
  const [uninstalling, setUninstalling] = useState(false);
  const [confirmUninstall, setConfirmUninstall] = useState(false);
  // Wallet export modal state
  const [showExportModal, setShowExportModal] = useState(false);
  const [walletAddress, setWalletAddress] = useState("");
  const [exportedKey, setExportedKey] = useState("");
  const [keyCopied, setKeyCopied] = useState(false);
  const [keySaved, setKeySaved] = useState(false);
  const [exporting, setExporting] = useState(false);

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
        <div className="flex items-center justify-center gap-3">
          <span className="text-3xl font-bold uppercase tracking-[0.2em] text-glow-pink">Install</span>
          <img src={logoImg} alt="AgentX" className="h-7 drop-shadow-[0_0_8px_rgba(255,0,128,0.4)]" />
        </div>
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

      {installed && (
        <NeonCard>
          <div className="space-y-3">
            <p className="text-xs text-white/30 uppercase tracking-wide">Danger Zone</p>
            {!confirmUninstall && !showExportModal ? (
              <NeonButton
                variant="danger"
                size="md"
                className="w-full"
                onClick={async () => {
                  try {
                    const hasWallet = await window.go.main.InstallerService.HasWallet();
                    if (hasWallet) {
                      // Load wallet address for display
                      try {
                        const w = await window.go.main.WalletService.GetWallet();
                        setWalletAddress(w.address);
                      } catch { /* noop */ }
                      setExportedKey("");
                      setKeyCopied(false);
                      setKeySaved(false);
                      setShowExportModal(true);
                    } else {
                      setConfirmUninstall(true);
                    }
                  } catch {
                    setConfirmUninstall(true);
                  }
                }}
              >
                Uninstall AgentX
              </NeonButton>
            ) : showExportModal ? (
              /* ── Mandatory wallet export modal ── */
              <div className="space-y-4">
                <div className="bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-3 space-y-2">
                  <p className="text-sm font-bold text-red-400 uppercase tracking-wide">
                    Warning: Wallet Will Be Deleted
                  </p>
                  <p className="text-xs text-red-300/70">
                    Uninstalling will permanently delete your wallet private key.
                    You must export and save it to retain access to any on-chain assets.
                  </p>
                </div>

                {walletAddress && (
                  <div className="bg-black/30 border border-white/[0.06] rounded-lg px-4 py-2.5">
                    <div className="text-[10px] uppercase tracking-widest text-white/30 mb-1">Wallet Address</div>
                    <div className="font-mono text-sm text-neon-cyan/80 break-all">{walletAddress}</div>
                  </div>
                )}

                {!exportedKey ? (
                  <NeonButton
                    variant="danger"
                    size="md"
                    className="w-full"
                    disabled={exporting}
                    onClick={async () => {
                      setExporting(true);
                      try {
                        const key = await window.go.main.WalletService.ExportPrivateKey();
                        setExportedKey(key);
                      } catch (e: any) {
                        showToast(`Export failed: ${e}`, "error");
                      } finally {
                        setExporting(false);
                      }
                    }}
                  >
                    {exporting ? "Decrypting..." : "Export Private Key"}
                  </NeonButton>
                ) : (
                  <div className="space-y-3">
                    <div className="bg-black/50 border border-red-500/20 rounded-lg px-4 py-3 space-y-2">
                      <div className="text-[10px] uppercase tracking-widest text-red-400/60 font-bold">Private Key</div>
                      <div className="font-mono text-xs text-white/80 break-all select-all leading-relaxed bg-black/40 rounded px-3 py-2">
                        {exportedKey}
                      </div>
                      <button
                        onClick={() => {
                          navigator.clipboard.writeText(exportedKey);
                          setKeyCopied(true);
                          showToast("Private key copied!", "success");
                        }}
                        className={`text-xs uppercase tracking-widest px-3 py-1.5 rounded border transition-all ${
                          keyCopied
                            ? "text-neon-green/80 border-neon-green/30 bg-neon-green/10"
                            : "text-white/50 border-white/10 hover:border-white/30 hover:text-white/80"
                        }`}
                      >
                        {keyCopied ? "✓ Copied" : "Copy to Clipboard"}
                      </button>
                    </div>

                    <label className="flex items-center gap-3 cursor-pointer px-1">
                      <input
                        type="checkbox"
                        checked={keySaved}
                        onChange={(e) => setKeySaved(e.target.checked)}
                        className="w-4 h-4 rounded border-white/20 bg-black/40 accent-neon-pink"
                      />
                      <span className="text-xs text-white/60">I have saved my private key in a secure location</span>
                    </label>

                    <div className="flex gap-3">
                      <NeonButton
                        variant="danger"
                        size="md"
                        className="flex-1"
                        disabled={!keySaved || uninstalling}
                        onClick={async () => {
                          setUninstalling(true);
                          try {
                            await window.go.main.InstallerService.FullUninstall();
                            showToast("AgentX uninstalled. Closing app...", "success");
                            setTimeout(() => window.runtime.Quit(), 1500);
                          } catch (e: any) {
                            showToast(`Uninstall failed: ${e}`, "error");
                            setUninstalling(false);
                          }
                        }}
                      >
                        {uninstalling ? "Uninstalling..." : "Proceed with Uninstall"}
                      </NeonButton>
                      <NeonButton
                        variant="ghost"
                        size="md"
                        className="flex-1"
                        onClick={() => {
                          setShowExportModal(false);
                          setExportedKey("");
                          setKeyCopied(false);
                          setKeySaved(false);
                        }}
                      >
                        Cancel
                      </NeonButton>
                    </div>
                  </div>
                )}

                {!exportedKey && (
                  <NeonButton
                    variant="ghost"
                    size="md"
                    className="w-full"
                    onClick={() => {
                      setShowExportModal(false);
                      setExportedKey("");
                    }}
                  >
                    Cancel
                  </NeonButton>
                )}
              </div>
            ) : (
              /* ── Simple confirmation (no wallet) ── */
              <div className="space-y-3">
                <p className="text-sm text-red-400">
                  This will remove the gateway binary, service, and all data (~/.agentx). Are you sure?
                </p>
                <div className="flex gap-3">
                  <NeonButton
                    variant="danger"
                    size="md"
                    className="flex-1"
                    disabled={uninstalling}
                    onClick={async () => {
                      setUninstalling(true);
                      try {
                        await window.go.main.InstallerService.FullUninstall();
                        showToast("AgentX uninstalled. Closing app...", "success");
                        setTimeout(() => window.runtime.Quit(), 1500);
                      } catch (e: any) {
                        showToast(`Uninstall failed: ${e}`, "error");
                        setUninstalling(false);
                      }
                    }}
                  >
                    {uninstalling ? "Uninstalling..." : "Yes, Uninstall"}
                  </NeonButton>
                  <NeonButton
                    variant="ghost"
                    size="md"
                    className="flex-1"
                    onClick={() => setConfirmUninstall(false)}
                  >
                    Cancel
                  </NeonButton>
                </div>
              </div>
            )}
          </div>
        </NeonCard>
      )}
    </div>
  );
}
