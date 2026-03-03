import { useRef, useCallback } from "react";

interface Props {
  children: React.ReactNode;
  className?: string;
  color?: string;
}

export default function GlowCard({
  children,
  className = "",
  color = "rgba(255, 0, 146, 0.4)",
}: Props) {
  const cardRef = useRef<HTMLDivElement>(null);

  const handleMouseMove = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      const el = cardRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      el.style.setProperty("--glow-x", `${x}px`);
      el.style.setProperty("--glow-y", `${y}px`);
    },
    [],
  );

  const handleMouseLeave = useCallback(() => {
    const el = cardRef.current;
    if (!el) return;
    el.style.removeProperty("--glow-x");
    el.style.removeProperty("--glow-y");
  }, []);

  return (
    <div
      ref={cardRef}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      className={`relative overflow-hidden rounded-xl ${className}`}
      style={
        {
          "--glow-color": color,
        } as React.CSSProperties
      }
    >
      {/* Glow overlay — follows cursor */}
      <div
        className="absolute inset-0 pointer-events-none opacity-0 transition-opacity duration-200"
        style={{
          background:
            "radial-gradient(300px circle at var(--glow-x, 50%) var(--glow-y, 50%), var(--glow-color), transparent 70%)",
          opacity: "var(--glow-x, none) != none ? 1 : 0",
        }}
      />
      {/* Always show a subtle glow when cursor is present */}
      <div
        className="absolute inset-0 pointer-events-none"
        style={{
          background:
            "radial-gradient(300px circle at var(--glow-x, -999px) var(--glow-y, -999px), var(--glow-color), transparent 70%)",
          opacity: 0.6,
        }}
      />
      <div className="relative z-10">{children}</div>
    </div>
  );
}
