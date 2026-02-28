import { useState, useRef, useEffect } from "react";

interface Option {
  id: string;
  label: string;
  sublabel?: string;
  badge?: string;
}

interface Props {
  label?: string;
  placeholder?: string;
  options: Option[];
  value: string | null;
  onChange: (id: string) => void;
}

export default function SearchableSelect({ label, placeholder = "Search...", options, value, onChange }: Props) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const ref = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const selected = options.find((o) => o.id === value);

  const filtered = options.filter(
    (o) =>
      o.label.toLowerCase().includes(query.toLowerCase()) ||
      (o.sublabel && o.sublabel.toLowerCase().includes(query.toLowerCase()))
  );

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const handleSelect = (id: string) => {
    onChange(id);
    setQuery("");
    setOpen(false);
  };

  return (
    <div ref={ref} className="relative">
      {label && (
        <label className="block text-xs text-white/50 mb-1.5 uppercase tracking-widest font-medium">
          {label}
        </label>
      )}
      <button
        type="button"
        onClick={() => {
          setOpen(!open);
          setTimeout(() => inputRef.current?.focus(), 50);
        }}
        className={`w-full flex items-center justify-between bg-white/[0.04] border-2 rounded-lg px-3 py-2.5 text-sm text-left transition-all ${
          open
            ? "border-neon-pink/50 shadow-neon-pink"
            : "border-neon-purple/20 hover:border-neon-pink/30"
        }`}
      >
        <span className={selected ? "text-white" : "text-white/30"}>
          {selected ? selected.label : "Select..."}
        </span>
        <span className={`text-white/30 text-xs transition-transform ${open ? "rotate-180" : ""}`}>
          ▼
        </span>
      </button>

      {open && (
        <div className="absolute z-50 mt-1.5 w-full bg-[#0d0b18] border-2 border-neon-pink/30 rounded-lg shadow-neon-pink overflow-hidden">
          <div className="p-2 border-b border-white/5">
            <input
              ref={inputRef}
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={placeholder}
              className="w-full bg-white/[0.04] border border-neon-purple/20 rounded-md px-3 py-2 text-sm text-white placeholder-white/25 focus:outline-none focus:border-neon-pink/40"
            />
          </div>
          <div className="max-h-52 overflow-y-auto">
            {filtered.length === 0 ? (
              <div className="px-3 py-4 text-xs text-white/25 text-center uppercase tracking-widest">
                No results
              </div>
            ) : (
              filtered.map((o) => (
                <button
                  key={o.id}
                  onClick={() => handleSelect(o.id)}
                  className={`w-full flex items-center justify-between px-3 py-2.5 text-left transition-all ${
                    value === o.id
                      ? "bg-neon-pink/10 text-neon-pink"
                      : "text-white/60 hover:bg-white/5 hover:text-white/90"
                  }`}
                >
                  <div>
                    <p className="text-sm font-medium">{o.label}</p>
                    {o.sublabel && (
                      <p className="text-xs text-white/25 font-mono">{o.sublabel}</p>
                    )}
                  </div>
                  {o.badge && (
                    <span className="text-[10px] text-neon-green/80 bg-neon-green/10 px-2 py-0.5 rounded border border-neon-green/20 uppercase tracking-widest font-bold">
                      {o.badge}
                    </span>
                  )}
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
