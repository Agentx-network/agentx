import type { ReactNode, ButtonHTMLAttributes } from "react";

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: "primary" | "ghost" | "danger";
  size?: "sm" | "md" | "lg";
}

export default function NeonButton({
  children,
  variant = "primary",
  size = "md",
  className = "",
  ...props
}: Props) {
  const base = "font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-neon-pink/50";

  const variants = {
    primary: "bg-gradient-to-r from-neon-pink to-neon-purple text-white hover:shadow-neon active:scale-[0.98]",
    ghost: "bg-white/5 text-white/70 hover:bg-white/10 hover:text-white border border-white/10",
    danger: "bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30",
  };

  const sizes = {
    sm: "px-3 py-1.5 text-xs",
    md: "px-4 py-2 text-sm",
    lg: "px-6 py-3 text-base",
  };

  return (
    <button className={`${base} ${variants[variant]} ${sizes[size]} ${className}`} {...props}>
      {children}
    </button>
  );
}
