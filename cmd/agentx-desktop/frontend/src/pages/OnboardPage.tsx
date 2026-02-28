import { useState, useEffect } from "react";
import type { ProviderOption } from "../lib/types";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import NeonInput from "../components/ui/NeonInput";
import SearchableSelect from "../components/ui/SearchableSelect";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
  onComplete: () => void;
}

export default function OnboardPage({ showToast, onComplete }: Props) {
  const [providers, setProviders] = useState<ProviderOption[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const load = async () => {
      try {
        const p = await window.go.main.ConfigService.GetAvailableProviders();
        setProviders(p);
      } catch { /* noop */ }
    };
    load();
  }, []);

  const selectedProvider = providers.find((p) => p.id === selected);

  const handleSetup = async () => {
    if (!selected) return;
    setSaving(true);
    try {
      await window.go.main.ConfigService.QuickSetupProvider(selected, apiKey);
      showToast("Provider configured!", "success");
      onComplete();
    } catch (e: any) {
      showToast(`Setup failed: ${e}`, "error");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="max-w-xl mx-auto space-y-6">
      <div className="text-center space-y-2">
        <h2 className="text-3xl font-bold uppercase tracking-[0.2em] text-glow-pink">Configure AI Provider</h2>
        <p className="text-white/35 text-sm">
          Choose an AI provider and enter your API key to power AgentX.
        </p>
      </div>

      <NeonCard>
        <div className="space-y-4">
          <SearchableSelect
            label="Provider"
            placeholder="Search by provider or model name..."
            options={providers.map((p) => ({
              id: p.id,
              label: `${p.name} — ${p.modelName}`,
              sublabel: p.model,
              badge: p.needsKey ? undefined : "Local",
            }))}
            value={selected}
            onChange={(id) => { setSelected(id); setApiKey(""); }}
          />
        </div>
      </NeonCard>

      {selectedProvider && (
        <NeonCard variant="pink" glow>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold uppercase tracking-widest text-white/90">
                {selectedProvider.name}
              </h3>
              <span className="text-xs text-white/25 font-mono">
                {selectedProvider.model}
              </span>
            </div>

            {selectedProvider.needsKey ? (
              <div className="space-y-3">
                <NeonInput
                  label="API Key"
                  value={apiKey}
                  onChange={setApiKey}
                  type="password"
                  placeholder={`Enter your ${selectedProvider.name} API key`}
                />
                {selectedProvider.keyURL && (
                  <a
                    href={selectedProvider.keyURL}
                    target="_blank"
                    rel="noopener"
                    className="text-xs text-neon-cyan hover:underline inline-block text-glow-cyan"
                  >
                    Get your API key at {selectedProvider.keyURL.replace("https://", "")} →
                  </a>
                )}
                <NeonButton
                  onClick={handleSetup}
                  disabled={!apiKey || saving}
                  size="lg"
                  className="w-full"
                >
                  {saving ? "Saving..." : "Save & Continue →"}
                </NeonButton>
              </div>
            ) : (
              <div className="space-y-3">
                <p className="text-sm text-white/40">
                  This provider runs locally — no API key required.
                </p>
                <NeonButton
                  onClick={handleSetup}
                  disabled={saving}
                  size="lg"
                  className="w-full"
                >
                  {saving ? "Saving..." : "Save & Continue →"}
                </NeonButton>
              </div>
            )}
          </div>
        </NeonCard>
      )}

      <button
        onClick={onComplete}
        className="block mx-auto text-xs text-white/25 hover:text-neon-pink/60 transition-colors uppercase tracking-widest"
      >
        Skip for now
      </button>
    </div>
  );
}
