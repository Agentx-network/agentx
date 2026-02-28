interface Props {
  label?: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  placeholder?: string;
}

export default function NeonInput({ label, value, onChange, type = "text", placeholder }: Props) {
  return (
    <div>
      {label && <label className="block text-xs text-white/50 mb-1">{label}</label>}
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/30 focus:outline-none focus:border-neon-pink/50 focus:ring-1 focus:ring-neon-pink/30 transition-colors"
      />
    </div>
  );
}
