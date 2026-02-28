import type { DownloadProgress } from "../lib/types";

interface Props {
  progress: DownloadProgress | null;
}

export default function DownloadProgressBar({ progress }: Props) {
  if (!progress) return null;

  const mb = (bytes: number) => (bytes / 1024 / 1024).toFixed(1);

  return (
    <div className="space-y-2">
      <div className="flex justify-between text-[10px] text-white/50 uppercase tracking-widest font-medium">
        <span>{mb(progress.downloaded)} MB / {mb(progress.total)} MB</span>
        <span>{progress.percent.toFixed(0)}%</span>
      </div>
      <div className="w-full h-2.5 bg-white/5 rounded-full overflow-hidden border border-neon-pink/20">
        <div
          className="h-full bg-gradient-to-r from-neon-pink to-neon-purple rounded-full transition-all duration-300 shadow-neon-pink"
          style={{ width: `${progress.percent}%` }}
        />
      </div>
    </div>
  );
}
