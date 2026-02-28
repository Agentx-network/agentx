import type { Page } from "../lib/types";

interface Props {
  currentPage: Page;
  onNavigate: (page: Page) => void;
  onRunWizard?: () => void;
}

const navItems: { id: Page; label: string; icon: string }[] = [
  { id: "dashboard", label: "Dashboard", icon: "📊" },
  { id: "chat", label: "Chat", icon: "💬" },
  { id: "config", label: "Config", icon: "⚙️" },
  { id: "installer", label: "Installer", icon: "📦" },
];

export default function Sidebar({ currentPage, onNavigate, onRunWizard }: Props) {
  return (
    <aside className="w-56 border-r-2 border-neon-pink/20 bg-bg-sidebar flex flex-col">
      <div className="p-5 border-b-2 border-neon-pink/20">
        <h1 className="text-xl font-bold text-white text-glow-pink uppercase tracking-widest">
          AgentX
        </h1>
        <p className="text-[10px] text-neon-pink/50 mt-1 uppercase tracking-[0.2em] font-medium">Desktop</p>
      </div>
      <nav className="flex-1 p-3 space-y-1">
        {navItems.map((item) => (
          <button
            key={item.id}
            onClick={() => onNavigate(item.id)}
            className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm uppercase tracking-widest transition-all ${
              currentPage === item.id
                ? "bg-neon-pink/15 text-neon-pink border-l-[3px] border-neon-pink shadow-glow-pink-sm"
                : "text-white/45 hover:text-white/80 hover:bg-white/5 border-l-[3px] border-transparent"
            }`}
          >
            <span>{item.icon}</span>
            <span className="font-bold text-xs">{item.label}</span>
          </button>
        ))}
      </nav>
      <div className="p-4 border-t-2 border-neon-pink/20 space-y-3">
        {onRunWizard && (
          <button
            onClick={onRunWizard}
            className="w-full text-[10px] text-neon-pink/40 hover:text-neon-pink hover:bg-neon-pink/10 uppercase tracking-widest font-bold py-2 px-3 rounded-lg border border-neon-pink/15 hover:border-neon-pink/30 transition-all"
          >
            Run Setup Wizard
          </button>
        )}
        <p className="text-[10px] text-white/25 uppercase tracking-widest">v0.1.0</p>
      </div>
    </aside>
  );
}
