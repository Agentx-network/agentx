import { useState, useEffect } from "react";
import type { AgentDefaults } from "../lib/types";
import NeonButton from "./ui/NeonButton";
import NeonInput from "./ui/NeonInput";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

export default function AgentSection({ showToast }: Props) {
  const [defaults, setDefaults] = useState<AgentDefaults | null>(null);

  const load = async () => {
    try {
      const d = await window.go.main.ConfigService.GetAgentDefaults();
      setDefaults(d);
    } catch { /* noop */ }
  };

  useEffect(() => { load(); }, []);

  const save = async () => {
    if (!defaults) return;
    try {
      await window.go.main.ConfigService.UpdateAgentDefaults(defaults);
      showToast("Agent defaults saved", "success");
    } catch (e: any) {
      showToast(e.toString(), "error");
    }
  };

  if (!defaults) return null;

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-white/70">Agent Defaults</h3>
      <div className="space-y-2">
        <NeonInput label="Workspace" value={defaults.workspace} onChange={(v) => setDefaults({ ...defaults, workspace: v })} />
        <NeonInput label="Model Name" value={defaults.model_name ?? defaults.model ?? ""} onChange={(v) => setDefaults({ ...defaults, model_name: v })} />
        <NeonInput label="Max Tokens" value={String(defaults.max_tokens)} onChange={(v) => setDefaults({ ...defaults, max_tokens: parseInt(v) || 0 })} />
        <NeonInput label="Temperature" value={defaults.temperature !== undefined ? String(defaults.temperature) : ""} onChange={(v) => setDefaults({ ...defaults, temperature: v ? parseFloat(v) : undefined })} />
        <NeonInput label="Max Tool Iterations" value={String(defaults.max_tool_iterations)} onChange={(v) => setDefaults({ ...defaults, max_tool_iterations: parseInt(v) || 0 })} />
        <NeonButton onClick={save} size="sm">Save Defaults</NeonButton>
      </div>
    </div>
  );
}
