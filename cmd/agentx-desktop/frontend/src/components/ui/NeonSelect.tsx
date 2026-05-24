interface Option {
  label: string;
  value: string;
}

interface Props {
  label?: string;
  value: string;
  options: Option[];
  onChange: (value: string) => void;
}

export default function NeonSelect({ label, value, options, onChange }: Props) {
  return (
    <div>
      {label && <label className="block text-xs text-white/50 mb-1.5 uppercase tracking-widest font-medium">{label}</label>}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full bg-white/[0.04] border-2 border-neon-purple/20 rounded-lg px-3 py-2.5 text-sm text-white focus:outline-none focus:border-neon-pink/50 focus:shadow-neon-pink transition-all appearance-none"
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value} className="bg-[#0d0b18] text-white">
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  );
}
