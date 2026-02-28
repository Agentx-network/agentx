import { useState, useEffect, useCallback } from "react";
import Sidebar from "./components/Sidebar";
import InstallerPage from "./pages/InstallerPage";
import OnboardPage from "./pages/OnboardPage";
import DashboardPage from "./pages/DashboardPage";
import ConfigPage from "./pages/ConfigPage";
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

export default function App() {
  const [mode, setMode] = useState<AppMode>("loading");
  const [page, setPage] = useState<Page>("installer");
  const [toast, setToast] = useState<{ message: string; type: "success" | "error" } | null>(null);

  const showToast = useCallback((message: string, type: "success" | "error" = "success") => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 3000);
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
      } else {
        setPage("dashboard");
        setMode("app");
      }
    } catch (e) {
      console.error("GetSetupState failed:", e);
      setPage("installer");
      setMode("wizard");
    }
  }, []);

  useEffect(() => {
    detectStep();
  }, [detectStep]);

  const onStepComplete = useCallback(() => {
    detectStep();
  }, [detectStep]);

  // Loading
  if (mode === "loading") {
    return (
      <div className="flex items-center justify-center h-screen bg-bg">
        <div className="text-center space-y-4">
          <h1 className="text-2xl font-bold bg-gradient-to-r from-neon-pink via-neon-purple to-neon-cyan bg-clip-text text-transparent">
            AgentX
          </h1>
          <p className="text-white/40 text-sm">Loading...</p>
        </div>
      </div>
    );
  }

  // Wizard mode — no sidebar, step indicator
  if (mode === "wizard") {
    const step = page === "installer" ? 1 : 2;
    return (
      <div className="flex flex-col h-screen bg-bg">
        <header className="flex items-center justify-between px-8 py-5 border-b border-white/10">
          <h1 className="text-xl font-bold bg-gradient-to-r from-neon-pink via-neon-purple to-neon-cyan bg-clip-text text-transparent">
            AgentX Setup
          </h1>
          <div className="flex items-center gap-3">
            <StepDot active={step >= 1} label="1. Install" />
            <div className="w-8 h-px bg-white/20" />
            <StepDot active={step >= 2} label="2. Configure" />
            <div className="w-8 h-px bg-white/20" />
            <StepDot active={false} label="3. Dashboard" />
          </div>
        </header>
        <main className="flex-1 overflow-y-auto p-8">
          {page === "installer" && (
            <InstallerPage showToast={showToast} onComplete={onStepComplete} />
          )}
          {page === "onboard" && (
            <OnboardPage showToast={showToast} onComplete={onStepComplete} />
          )}
        </main>
        {toast && <Toast message={toast.message} type={toast.type} />}
      </div>
    );
  }

  // Normal app mode — sidebar + pages
  return (
    <div className="flex h-screen bg-bg">
      <Sidebar currentPage={page} onNavigate={setPage} />
      <main className="flex-1 overflow-y-auto p-6">
        {page === "dashboard" && <DashboardPage showToast={showToast} />}
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
        className={`w-2.5 h-2.5 rounded-full transition-colors ${
          active ? "bg-neon-pink shadow-neon" : "bg-white/20"
        }`}
      />
      <span className={`text-xs ${active ? "text-white/80" : "text-white/30"}`}>
        {label}
      </span>
    </div>
  );
}
