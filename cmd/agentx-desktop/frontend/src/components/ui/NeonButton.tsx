import type { ReactNode, ButtonHTMLAttributes } from "react";

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: "primary" | "ghost" | "green" | "danger";
  size?: "sm" | "md" | "lg";
}

export default function NeonButton({
  children,
  variant = "primary",
  size = "md",
  className = "",
  ...props
}: Props) {
  const base = "font-bold uppercase tracking-wider rounded-lg transition-all duration-200 focus:outline-none disabled:opacity-40 disabled:cursor-not-allowed";

  const variants = {
    primary:
      "bg-neon-pink text-white hover:shadow-neon-pink active:scale-[0.97] border border-neon-pink/60",
    green:
      "bg-neon-green text-black hover:shadow-neon-green active:scale-[0.97] border border-neon-green/60",
    ghost:
      "bg-transparent text-neon-pink border-2 border-neon-pink/40 hover:bg-neon-pink/10 hover:border-neon-pink/60 hover:shadow-neon-pink",
    danger:
      "bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/40",
  };

  const sizes = {
    sm: "px-3 py-1.5 text-xs",
    md: "px-5 py-2.5 text-sm",
    lg: "px-7 py-3 text-base",
  };

  return (
    <button className={`${base} ${variants[variant]} ${sizes[size]} ${className}`} {...props}>
      {children}
    </button>
  );
}
