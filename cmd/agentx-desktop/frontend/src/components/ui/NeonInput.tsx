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
      {label && <label className="block text-xs text-white/50 mb-1.5 uppercase tracking-widest font-medium">{label}</label>}
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full bg-white/[0.04] border-2 border-neon-purple/20 rounded-lg px-3 py-2.5 text-sm text-white placeholder-white/25 focus:outline-none focus:border-neon-pink/50 focus:shadow-neon-pink transition-all"
      />
    </div>
  );
}
