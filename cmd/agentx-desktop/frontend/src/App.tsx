import { useState, useEffect, useCallback } from "react";
import Sidebar from "./components/Sidebar";
import InstallerPage from "./pages/InstallerPage";
import OnboardPage from "./pages/OnboardPage";
import ChannelSetupPage from "./pages/ChannelSetupPage";
import AgentSetupPage from "./pages/AgentSetupPage";
import DashboardPage from "./pages/DashboardPage";
import ConfigPage from "./pages/ConfigPage";
import ChatPage from "./pages/ChatPage";
import { Toast } from "./components/ui/Toast";
import type { Page, SetupState, ChatMessage } from "./lib/types";
import agentHero from "./assets/agent-hero.gif";
import posterBg from "./assets/poster-bg.jpg";
import logo from "./assets/logo.png";

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetAppInfo(): Promise<any>;
          ConfigExists(): Promise<boolean>;
          GetConfigPath(): Promise<string>;
          GetSetupState(): Promise<SetupState>;
        };
        InstallerService: {
          DetectPlatform(): Promise<any>;
          GetLatestRelease(): Promise<string>;
          InstallBinary(): Promise<void>;
          InstallService(): Promise<void>;
          UninstallService(): Promise<void>;
          FullUninstall(): Promise<void>;
          IsServiceRunning(): Promise<boolean>;
        };
        DashboardService: {
          GetStatus(): Promise<any>;
          GetLogs(lines: number): Promise<string>;
          StartGateway(): Promise<void>;
          StopGateway(): Promise<void>;
          RestartGateway(): Promise<void>;
        };
        ConfigService: {
          GetConfig(): Promise<any>;
          SaveConfig(cfg: any): Promise<void>;
          GetModelList(): Promise<any[]>;
          AddModel(model: any): Promise<void>;
          UpdateModel(index: number, model: any): Promise<void>;
          RemoveModel(index: number): Promise<void>;
          SetChannelEnabled(channel: string, enabled: boolean): Promise<void>;
          GetAgentDefaults(): Promise<any>;
          UpdateAgentDefaults(defaults: any): Promise<void>;
          GetAvailableProviders(): Promise<any[]>;
          QuickSetupProvider(providerID: string, apiKey: string): Promise<void>;
          QuickSetupChannel(channel: string, token: string): Promise<void>;
        };
        ChatService: {
          SendMessage(message: string, sessionKey: string): Promise<{ response: string }>;
          IsGatewayReachable(): Promise<boolean>;
          GetChatHistory(sessionKey: string): Promise<{ role: string; content: string; timestamp: number }[]>;
        };
        AgentSetupService: {
          GetBootstrapFiles(): Promise<{ name: string; path: string; content: string; exists: boolean }[]>;
          SaveBootstrapFile(name: string, content: string): Promise<void>;
          CreateDefaultBootstrapFiles(): Promise<void>;
          ListSkills(): Promise<{ name: string; source: string; description: string; path: string }[]>;
          GetSkillContent(name: string): Promise<string>;
          InstallSkill(repo: string): Promise<void>;
          RemoveSkill(name: string): Promise<void>;
          InstallBuiltinSkills(): Promise<void>;
          SearchSkills(query: string): Promise<{ slug: string; displayName: string; summary: string; version: string; registry: string; score: number }[]>;
          InstallFromRegistry(slug: string): Promise<void>;
        };
      };
    };
    runtime: {
      EventsOn(event: string, callback: (...args: any[]) => void): void;
      EventsOff(event: string): void;
      Quit(): void;
    };
  }
}

type AppMode = "loading" | "wizard" | "app";

const wizardSteps = [
  { key: "installer", label: "Install" },
  { key: "onboard", label: "Provider" },
  { key: "channels", label: "Channel" },
  { key: "agent", label: "Agent" },
  { key: "dashboard", label: "Dashboard" },
] as const;

export default function App() {
  const [mode, setMode] = useState<AppMode>("loading");
  const [page, setPage] = useState<Page>("installer");
  const [toast, setToast] = useState<{ message: string; type: "success" | "error" } | null>(null);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [appVersion, setAppVersion] = useState("");

  const showToast = useCallback((message: string, type: "success" | "error" = "success") => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 3000);
  }, []);

  // Load chat history from disk on startup
  useEffect(() => {
    (async () => {
      try {
        const history = await window.go.main.ChatService.GetChatHistory("");
        if (history && history.length > 0) {
          const msgs: ChatMessage[] = history.map((h, i) => ({
            id: `hist-${i}`,
            role: h.role as "user" | "assistant",
            content: h.content,
            timestamp: h.timestamp,
          }));
          setChatMessages(msgs);
        }
      } catch {
        // No history yet — ignore
      }
    })();
  }, []);

  // Fetch desktop app version for sidebar display
  useEffect(() => {
    (async () => {
      try {
        const info = await window.go.main.App.GetAppInfo();
        if (info.version && info.version !== "dev") {
          setAppVersion("v" + info.version);
        }
      } catch { /* noop */ }
    })();
  }, []);

  // Wizard step order (excluding final "dashboard" marker)
  const wizardPageOrder: Page[] = ["installer", "onboard", "channels", "agent"];

  const finishWizard = useCallback(async () => {
    setPage("dashboard");
    try {
      await window.go.main.InstallerService.InstallService();
    } catch { /* best-effort */ }
    try {
      await window.go.main.DashboardService.StartGateway();
    } catch { /* dashboard will show status */ }
    setMode("app");
  }, []);

  const detectStep = useCallback(async () => {
    try {
      const state = await window.go.main.App.GetSetupState();
      console.log("SetupState:", JSON.stringify(state));

      if (!state.binaryInstalled) {
        setPage("installer");
        setMode("wizard");
      } else if (!state.hasApiKey) {
        setPage("onboard");
        setMode("wizard");
      } else if (!state.hasChannel) {
        setPage("channels");
        setMode("wizard");
      } else {
        // All core requirements met — skip to app mode
        await finishWizard();
      }
    } catch (e) {
      console.error("GetSetupState failed:", e);
      setPage("installer");
      setMode("wizard");
    }
  }, [finishWizard]);

  useEffect(() => {
    detectStep();
  }, [detectStep]);

  // Advance wizard sequentially — always shows every remaining step
  const onStepComplete = useCallback(async () => {
    const currentIdx = wizardPageOrder.indexOf(page as Page);
    if (currentIdx >= 0 && currentIdx < wizardPageOrder.length - 1) {
      // Move to next wizard step
      setPage(wizardPageOrder[currentIdx + 1]);
    } else {
      // Last wizard step done — finish setup
      await finishWizard();
    }
  }, [page, finishWizard]);

  // Subtle ambient poster background — shared across all screens
  const ambientBg = (
    <div className="absolute inset-0 pointer-events-none overflow-hidden">
      <img src={posterBg} alt="" className="w-full h-full object-cover opacity-[0.12]" />
    </div>
  );

  if (mode === "loading") {
    return (
      <div className="relative flex items-end justify-center h-screen overflow-hidden">
        {/* Poster background — full opacity for loading */}
        <img
          src={posterBg}
          alt=""
          className="absolute inset-0 w-full h-full object-cover"
        />
        {/* Dark gradient overlay for blending */}
        <div className="absolute inset-0 bg-gradient-to-t from-[#0a0a12] via-[#0a0a12]/60 to-transparent" />
        <div className="absolute inset-0 bg-gradient-to-b from-[#0a0a12]/50 to-transparent h-1/3" />
        {/* Content at bottom */}
        <div className="relative z-10 text-center space-y-4 pb-12">
          <img
            src={agentHero}
            alt=""
            className="w-20 h-20 mx-auto rounded-full border-2 border-neon-pink/40 shadow-[0_0_40px_rgba(255,0,128,0.35)] animate-pulse"
          />
          <div className="space-y-3">
            <img src={logo} alt="AgentX" className="h-8 mx-auto drop-shadow-[0_0_12px_rgba(255,0,128,0.5)]" />
            <p className="text-neon-pink/60 text-sm uppercase tracking-widest animate-pulse">Loading...</p>
          </div>
        </div>
      </div>
    );
  }

  if (mode === "wizard") {
    const currentIdx = wizardSteps.findIndex((s) => s.key === page);

    return (
      <div className="relative flex flex-col h-screen bg-bg">
        {ambientBg}
        <header className="relative flex items-center justify-between px-8 py-5 border-b-2 border-neon-pink/20">
          <div className="flex items-center gap-3">
            <img src={logo} alt="AgentX" className="h-5 drop-shadow-[0_0_8px_rgba(255,0,128,0.4)]" />
            <span className="text-xl font-bold text-white text-glow-pink uppercase tracking-[0.2em]">Setup</span>
          </div>
          <div className="flex items-center gap-2">
            {wizardSteps.map((s, i) => (
              <div key={s.key} className="flex items-center gap-2">
                {i > 0 && <div className="w-6 h-px bg-neon-pink/20" />}
                <StepDot active={i <= currentIdx} label={`${i + 1}. ${s.label}`} />
              </div>
            ))}
          </div>
        </header>
        <main className="relative flex-1 overflow-y-auto p-8">
          {page === "installer" && (
            <InstallerPage showToast={showToast} onComplete={onStepComplete} />
          )}
          {page === "onboard" && (
            <OnboardPage showToast={showToast} onComplete={onStepComplete} />
          )}
          {page === "channels" && (
            <ChannelSetupPage showToast={showToast} onComplete={onStepComplete} />
          )}
          {page === "agent" && (
            <AgentSetupPage showToast={showToast} onComplete={onStepComplete} />
          )}
        </main>
        {toast && <Toast message={toast.message} type={toast.type} />}
      </div>
    );
  }

  return (
    <div className="relative flex h-screen bg-bg">
      {ambientBg}
      <Sidebar currentPage={page} onNavigate={setPage} onRunWizard={() => { setPage("installer"); setMode("wizard"); }} version={appVersion} />
      <main className={`relative flex-1 p-6 ${page === "chat" ? "overflow-hidden flex flex-col" : "overflow-y-auto"}`}>
        {page === "dashboard" && <DashboardPage showToast={showToast} />}
        {page === "chat" && <ChatPage showToast={showToast} messages={chatMessages} setMessages={setChatMessages} />}
        {page === "config" && <ConfigPage showToast={showToast} />}
        {page === "installer" && <InstallerPage showToast={showToast} onComplete={onStepComplete} />}
      </main>
      {toast && <Toast message={toast.message} type={toast.type} />}
    </div>
  );
}

function StepDot({ active, label }: { active: boolean; label: string }) {
  return (
    <div className="flex items-center gap-2">
      <div
        className={`w-2.5 h-2.5 rounded-full transition-all ${
          active ? "bg-neon-pink shadow-glow-pink-sm" : "bg-white/15"
        }`}
      />
      <span className={`text-[10px] uppercase tracking-widest font-medium ${active ? "text-neon-pink" : "text-white/25"}`}>
        {label}
      </span>
    </div>
  );
}
