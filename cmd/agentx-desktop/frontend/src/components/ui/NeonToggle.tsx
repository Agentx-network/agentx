import { useState, useCallback } from "react";

interface Props {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
}

export default function NeonToggle({ checked, onChange, label }: Props) {
  const [sparking, setSparking] = useState(false);

  const handleToggle = useCallback(() => {
    onChange(!checked);
    setSparking(true);
    setTimeout(() => setSparking(false), 400);
  }, [checked, onChange]);

  return (
    <label className="flex items-center gap-2.5 cursor-pointer">
      <div
        className={`relative w-10 h-5.5 rounded-full transition-all ${
          checked ? "bg-neon-pink/30 shadow-glow-pink-sm" : "bg-white/10"
        }`}
        style={{ width: 40, height: 22 }}
        onClick={handleToggle}
      >
        <div
          className={`absolute top-[3px] w-4 h-4 rounded-full transition-all ${
            checked ? "left-[20px] bg-neon-pink shadow-glow-pink-sm" : "left-[3px] bg-white/40"
          }`}
        />
        {/* Spark effect */}
        {sparking && (
          <div
            className="absolute top-1/2 -translate-y-1/2 w-3 h-3 rounded-full pointer-events-none"
            style={{
              left: checked ? 22 : 5,
              background: "radial-gradient(circle, rgba(255,0,146,0.8), transparent 70%)",
              animation: "toggle-spark 0.4s ease-out forwards",
            }}
          />
        )}
      </div>
      {label && <span className="text-sm text-white/70">{label}</span>}
    </label>
  );
}
