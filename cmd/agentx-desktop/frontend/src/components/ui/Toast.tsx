interface Props {
  message: string;
  type: "success" | "error";
}

export function Toast({ message, type }: Props) {
  return (
    <div className="fixed bottom-6 right-6 z-50 animate-in slide-in-from-bottom-4">
      <div
        className={`px-5 py-3 rounded-lg text-sm font-bold uppercase tracking-wide border-2 ${
          type === "success"
            ? "bg-neon-green/10 text-neon-green border-neon-green/40 shadow-neon-green"
            : "bg-red-500/15 text-red-400 border-red-500/40 shadow-[0_0_15px_rgba(255,50,50,0.3)]"
        }`}
      >
        {message}
      </div>
    </div>
  );
}
