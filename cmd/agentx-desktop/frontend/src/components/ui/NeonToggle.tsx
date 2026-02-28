interface Props {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
}

export default function NeonToggle({ checked, onChange, label }: Props) {
  return (
    <label className="flex items-center gap-2.5 cursor-pointer">
      <div
        className={`relative w-10 h-5.5 rounded-full transition-all ${
          checked ? "bg-neon-pink/30 shadow-glow-pink-sm" : "bg-white/10"
        }`}
        style={{ width: 40, height: 22 }}
        onClick={() => onChange(!checked)}
      >
        <div
          className={`absolute top-[3px] w-4 h-4 rounded-full transition-all ${
            checked ? "left-[20px] bg-neon-pink shadow-glow-pink-sm" : "left-[3px] bg-white/40"
          }`}
        />
      </div>
      {label && <span className="text-sm text-white/70">{label}</span>}
    </label>
  );
}
