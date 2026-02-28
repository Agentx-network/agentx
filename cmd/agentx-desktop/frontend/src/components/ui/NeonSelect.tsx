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
      {label && <label className="block text-xs text-white/50 mb-1">{label}</label>}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-neon-pink/50 focus:ring-1 focus:ring-neon-pink/30 transition-colors appearance-none"
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value} className="bg-bg text-white">
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  );
}
