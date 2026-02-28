import type { ReactNode } from "react";

interface Props {
  children: ReactNode;
  title?: string;
  collapsible?: boolean;
  className?: string;
}

import { useState } from "react";

export default function NeonCard({ children, title, collapsible = false, className = "" }: Props) {
  const [open, setOpen] = useState(true);

  return (
    <div className={`glass-card ${className}`}>
      {title && (
        <div
          className={`flex items-center justify-between px-5 py-3 border-b border-white/5 ${
            collapsible ? "cursor-pointer hover:bg-white/[0.02]" : ""
          }`}
          onClick={() => collapsible && setOpen(!open)}
        >
          <h3 className="text-sm font-medium text-white/80">{title}</h3>
          {collapsible && (
            <span className={`text-white/30 text-xs transition-transform ${open ? "rotate-180" : ""}`}>
              ▼
            </span>
          )}
        </div>
      )}
      {open && <div className="p-5">{children}</div>}
    </div>
  );
}
