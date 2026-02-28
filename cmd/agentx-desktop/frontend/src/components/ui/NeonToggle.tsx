interface Props {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
}

export default function NeonToggle({ checked, onChange, label }: Props) {
  return (
    <label className="flex items-center gap-2 cursor-pointer">
      <div
        className={`relative w-9 h-5 rounded-full transition-colors ${
          checked ? "bg-neon-pink/40" : "bg-white/10"
        }`}
        onClick={() => onChange(!checked)}
      >
        <div
          className={`absolute top-0.5 w-4 h-4 rounded-full transition-all ${
            checked ? "left-[18px] bg-neon-pink" : "left-0.5 bg-white/40"
          }`}
        />
      </div>
      {label && <span className="text-sm text-white/70">{label}</span>}
    </label>
  );
}
