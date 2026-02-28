import { useState, useEffect } from "react";
import type { ModelConfig } from "../lib/types";
import NeonButton from "./ui/NeonButton";
import NeonInput from "./ui/NeonInput";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

export default function ProvidersSection({ showToast }: Props) {
  const [models, setModels] = useState<ModelConfig[]>([]);
  const [editing, setEditing] = useState<number | null>(null);
  const [draft, setDraft] = useState<Partial<ModelConfig>>({});

  const load = async () => {
    try {
      const list = await window.go.main.ConfigService.GetModelList();
      setModels(list ?? []);
    } catch { /* noop */ }
  };

  useEffect(() => { load(); }, []);

  const startEdit = (i: number) => {
    setEditing(i);
    setDraft(models[i]);
  };

  const save = async () => {
    if (editing === null) return;
    try {
      await window.go.main.ConfigService.UpdateModel(editing, draft as any);
      showToast("Model updated", "success");
      setEditing(null);
      load();
    } catch (e: any) {
      showToast(e.toString(), "error");
    }
  };

  const remove = async (i: number) => {
    try {
      await window.go.main.ConfigService.RemoveModel(i);
      showToast("Model removed", "success");
      load();
    } catch (e: any) {
      showToast(e.toString(), "error");
    }
  };

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-white/70">Models</h3>
      <div className="space-y-2">
        {models.map((m, i) => (
          <div key={i} className="glass-card p-3">
            {editing === i ? (
              <div className="space-y-2">
                <NeonInput label="Model Name" value={draft.model_name ?? ""} onChange={(v) => setDraft({ ...draft, model_name: v })} />
                <NeonInput label="Model" value={draft.model ?? ""} onChange={(v) => setDraft({ ...draft, model: v })} />
                <NeonInput label="API Base" value={draft.api_base ?? ""} onChange={(v) => setDraft({ ...draft, api_base: v })} />
                <NeonInput label="API Key" value={draft.api_key ?? ""} onChange={(v) => setDraft({ ...draft, api_key: v })} type="password" />
                <div className="flex gap-2">
                  <NeonButton onClick={save} size="sm">Save</NeonButton>
                  <NeonButton onClick={() => setEditing(null)} variant="ghost" size="sm">Cancel</NeonButton>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-between">
                <div>
                  <span className="text-sm text-white">{m.model_name}</span>
                  <span className="text-xs text-white/30 ml-2">{m.model}</span>
                  {m.api_key && <span className="text-xs text-neon-cyan ml-2">●</span>}
                </div>
                <div className="flex gap-2">
                  <button onClick={() => startEdit(i)} className="text-xs text-white/50 hover:text-white">Edit</button>
                  <button onClick={() => remove(i)} className="text-xs text-red-400/50 hover:text-red-400">Remove</button>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
