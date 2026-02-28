interface Props {
  running: boolean;
  label?: string;
}

export default function StatusBadge({ running, label }: Props) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${
        running
          ? "bg-green-500/20 text-green-400 border border-green-500/30"
          : "bg-red-500/20 text-red-400 border border-red-500/30"
      }`}
    >
      <span
        className={`w-2 h-2 rounded-full ${
          running ? "bg-green-400 animate-pulse-neon" : "bg-red-400"
        }`}
      />
      {label ?? (running ? "Running" : "Stopped")}
    </span>
  );
}
