import type { ReactNode } from "react";
import { useState } from "react";

interface Props {
  children: ReactNode;
  title?: string;
  collapsible?: boolean;
  variant?: "pink" | "green" | "cyan" | "purple";
  glow?: boolean;
  className?: string;
}

const variantClasses: Record<string, string> = {
  pink: "card-pink",
  green: "card-green",
  cyan: "card-cyan",
  purple: "card-purple",
};

export default function NeonCard({ children, title, collapsible = false, variant, glow = false, className = "" }: Props) {
  const [open, setOpen] = useState(true);
  const vClass = variant ? variantClasses[variant] : "";
  const glowClass = glow ? "animate-border-glow" : "";

  return (
    <div className={`glass-card ${vClass} ${glowClass} ${className}`}>
      {title && (
        <div
          className={`flex items-center justify-between px-5 py-3.5 border-b border-white/[0.06] ${
            collapsible ? "cursor-pointer hover:bg-white/[0.02]" : ""
          }`}
          onClick={() => collapsible && setOpen(!open)}
        >
          <h3 className="text-sm font-bold uppercase tracking-widest text-white/90">{title}</h3>
          {collapsible && (
            <span className={`text-neon-pink/50 text-xs transition-transform ${open ? "rotate-180" : ""}`}>
              ▼
            </span>
          )}
        </div>
      )}
      {open && <div className="p-5">{children}</div>}
    </div>
  );
}
