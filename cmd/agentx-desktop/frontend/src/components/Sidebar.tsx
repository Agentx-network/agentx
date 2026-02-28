import type { Page } from "../lib/types";

interface Props {
  currentPage: Page;
  onNavigate: (page: Page) => void;
}

const navItems: { id: Page; label: string; icon: string }[] = [
  { id: "dashboard", label: "Dashboard", icon: "📊" },
  { id: "config", label: "Config", icon: "⚙️" },
  { id: "installer", label: "Installer", icon: "📦" },
];

export default function Sidebar({ currentPage, onNavigate }: Props) {
  return (
    <aside className="w-56 border-r border-white/10 bg-bg flex flex-col">
      <div className="p-5 border-b border-white/10">
        <h1 className="text-xl font-bold bg-gradient-to-r from-neon-pink via-neon-purple to-neon-cyan bg-clip-text text-transparent">
          AgentX
        </h1>
        <p className="text-xs text-white/40 mt-1">Desktop</p>
      </div>
      <nav className="flex-1 p-3 space-y-1">
        {navItems.map((item) => (
          <button
            key={item.id}
            onClick={() => onNavigate(item.id)}
            className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all ${
              currentPage === item.id
                ? "bg-white/10 text-white shadow-neon"
                : "text-white/50 hover:text-white/80 hover:bg-white/5"
            }`}
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>
      <div className="p-4 border-t border-white/10 text-xs text-white/30">
        v0.1.0
      </div>
    </aside>
  );
}
