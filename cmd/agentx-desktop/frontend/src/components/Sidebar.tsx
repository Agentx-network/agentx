import type { Page } from "../lib/types";
import agentHero from "../assets/agent-hero.gif";
import logo from "../assets/logo.png";

interface Props {
  currentPage: Page;
  onNavigate: (page: Page) => void;
  onRunWizard?: () => void;
  version?: string;
}

const navItems: { id: Page; label: string; icon: (active: boolean) => JSX.Element }[] = [
  {
    id: "dashboard",
    label: "Dashboard",
    icon: (a) => (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" className={a ? "text-neon-pink" : "text-current"}>
        <rect x="1" y="1" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
        <rect x="9" y="1" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
        <rect x="1" y="9" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
        <rect x="9" y="9" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
      </svg>
    ),
  },
  {
    id: "chat",
    label: "Chat",
    icon: (a) => (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" className={a ? "text-neon-pink" : "text-current"}>
        <path d="M2 3.5C2 2.67 2.67 2 3.5 2h9c.83 0 1.5.67 1.5 1.5v7c0 .83-.67 1.5-1.5 1.5H5.5L2 14.5V3.5z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
        <line x1="5" y1="6" x2="11" y2="6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        <line x1="5" y1="9" x2="9" y2="9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    id: "config",
    label: "Config",
    icon: (a) => (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" className={a ? "text-neon-pink" : "text-current"}>
        <path d="M6.86 1.5h2.28l.35 1.75.96.4 1.55-.95 1.61 1.61-.95 1.55.4.96 1.75.35v2.28l-1.75.35-.4.96.95 1.55-1.61 1.61-1.55-.95-.96.4-.35 1.75H6.86l-.35-1.75-.96-.4-1.55.95-1.61-1.61.95-1.55-.4-.96L1.19 9.14V6.86l1.75-.35.4-.96-.95-1.55L4 2.39l1.55.95.96-.4.35-1.84z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
        <circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.3" />
      </svg>
    ),
  },
  {
    id: "wallet",
    label: "Wallet",
    icon: (a) => (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" className={a ? "text-neon-pink" : "text-current"}>
        <rect x="1.5" y="3.5" width="13" height="10" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
        <path d="M1.5 6.5h13" stroke="currentColor" strokeWidth="1.5" />
        <circle cx="11.5" cy="9.5" r="1" fill="currentColor" />
        <path d="M4 3.5V2.5a1 1 0 011-1h6a1 1 0 011 1v1" stroke="currentColor" strokeWidth="1.5" />
      </svg>
    ),
  },
  {
    id: "installer",
    label: "Installer",
    icon: (a) => (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" className={a ? "text-neon-pink" : "text-current"}>
        <path d="M8 2v8m0 0l-3-3m3 3l3-3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M2 11v2.5c0 .28.22.5.5.5h11c.28 0 .5-.22.5-.5V11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    ),
  },
];

export default function Sidebar({ currentPage, onNavigate, onRunWizard, version }: Props) {
  return (
    <aside className="w-56 border-r-2 border-neon-pink/20 bg-bg-sidebar flex flex-col">
      <div className="p-5 border-b-2 border-neon-pink/20 flex items-center gap-3">
        <img src={agentHero} alt="" className="w-10 h-10 rounded-lg border border-neon-pink/30 shadow-[0_0_12px_rgba(255,0,128,0.2)]" />
        <div>
          <img src={logo} alt="AgentX" className="h-5 w-auto drop-shadow-[0_0_8px_rgba(255,0,128,0.4)]" />
          <p className="text-[11px] text-neon-pink/60 uppercase tracking-[0.2em] font-medium mt-1">Desktop</p>
        </div>
      </div>
      <nav className="flex-1 p-3 space-y-1">
        {navItems.map((item) => {
          const active = currentPage === item.id;
          return (
            <button
              key={item.id}
              onClick={() => onNavigate(item.id)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm uppercase tracking-widest transition-all ${
                active
                  ? "bg-neon-pink/15 text-neon-pink border-l-[3px] border-neon-pink shadow-glow-pink-sm"
                  : "text-white/55 hover:text-white/80 hover:bg-white/5 border-l-[3px] border-transparent"
              }`}
            >
              {item.icon(active)}
              <span className="font-bold text-xs">{item.label}</span>
            </button>
          );
        })}
      </nav>
      <div className="p-4 border-t-2 border-neon-pink/20 space-y-3">
        {onRunWizard && (
          <button
            onClick={onRunWizard}
            className="w-full text-[11px] text-neon-pink/60 hover:text-neon-pink hover:bg-neon-pink/10 uppercase tracking-widest font-bold py-2 px-3 rounded-lg border border-neon-pink/20 hover:border-neon-pink/30 transition-all"
          >
            Run Setup Wizard
          </button>
        )}
        <p className="text-[11px] text-white/40 uppercase tracking-widest">{version || ""}</p>
      </div>
    </aside>
  );
}
