interface Props {
  running: boolean;
  label?: string;
}

export default function StatusBadge({ running, label }: Props) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[10px] font-bold uppercase tracking-widest border-2 ${
        running
          ? "bg-neon-green/10 text-neon-green border-neon-green/30 shadow-glow-green-sm"
          : "bg-red-500/15 text-red-400 border-red-500/30"
      }`}
    >
      <span
        className={`w-2 h-2 rounded-full ${
          running ? "bg-neon-green animate-pulse-neon shadow-glow-green-sm" : "bg-red-400"
        }`}
      />
      {label ?? (running ? "Running" : "Stopped")}
    </span>
  );
}
