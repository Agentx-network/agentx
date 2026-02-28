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
import type { Page, SetupState } from "./lib/types";

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
        };
      };
    };
    runtime: {
      EventsOn(event: string, callback: (...args: any[]) => void): void;
      EventsOff(event: string): void;
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

  const showToast = useCallback((message: string, type: "success" | "error" = "success") => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 3000);
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

  if (mode === "loading") {
    return (
      <div className="flex items-center justify-center h-screen bg-bg">
        <div className="text-center space-y-4">
          <h1 className="text-3xl font-bold text-white text-glow-pink uppercase tracking-[0.3em]">
            AgentX
          </h1>
          <p className="text-neon-pink/40 text-sm uppercase tracking-widest">Loading...</p>
        </div>
      </div>
    );
  }

  if (mode === "wizard") {
    const currentIdx = wizardSteps.findIndex((s) => s.key === page);

    return (
      <div className="flex flex-col h-screen bg-bg">
        <header className="flex items-center justify-between px-8 py-5 border-b-2 border-neon-pink/20">
          <h1 className="text-xl font-bold text-white text-glow-pink uppercase tracking-[0.2em]">
            AgentX Setup
          </h1>
          <div className="flex items-center gap-2">
            {wizardSteps.map((s, i) => (
              <div key={s.key} className="flex items-center gap-2">
                {i > 0 && <div className="w-6 h-px bg-neon-pink/20" />}
                <StepDot active={i <= currentIdx} label={`${i + 1}. ${s.label}`} />
              </div>
            ))}
          </div>
        </header>
        <main className="flex-1 overflow-y-auto p-8">
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
    <div className="flex h-screen bg-bg">
      <Sidebar currentPage={page} onNavigate={setPage} onRunWizard={() => { setPage("installer"); setMode("wizard"); }} />
      <main className="flex-1 overflow-y-auto p-6">
        {page === "dashboard" && <DashboardPage showToast={showToast} />}
        {page === "chat" && <ChatPage showToast={showToast} />}
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
