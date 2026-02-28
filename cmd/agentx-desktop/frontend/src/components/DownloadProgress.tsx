import type { DownloadProgress } from "../lib/types";

interface Props {
  progress: DownloadProgress | null;
}

export default function DownloadProgressBar({ progress }: Props) {
  if (!progress) return null;

  const mb = (bytes: number) => (bytes / 1024 / 1024).toFixed(1);

  return (
    <div className="space-y-2">
      <div className="flex justify-between text-xs text-white/60">
        <span>{mb(progress.downloaded)} MB / {mb(progress.total)} MB</span>
        <span>{progress.percent.toFixed(0)}%</span>
      </div>
      <div className="w-full h-2 bg-white/10 rounded-full overflow-hidden">
        <div
          className="h-full bg-gradient-to-r from-neon-pink to-neon-purple rounded-full transition-all duration-300"
          style={{ width: `${progress.percent}%` }}
        />
      </div>
    </div>
  );
}
