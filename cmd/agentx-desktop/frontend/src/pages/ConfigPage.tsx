import { useState, useEffect } from "react";
import type { ProviderOption } from "../lib/types";
import NeonCard from "../components/ui/NeonCard";
import NeonButton from "../components/ui/NeonButton";
import NeonInput from "../components/ui/NeonInput";
import SearchableSelect from "../components/ui/SearchableSelect";
import ProvidersSection from "../components/ProvidersSection";
import ChannelsSection from "../components/ChannelsSection";
import AgentSection from "../components/AgentSection";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

export default function ConfigPage({ showToast }: Props) {
  const [providers, setProviders] = useState<ProviderOption[]>([]);
  const [selectedProvider, setSelectedProvider] = useState("");
  const [apiKey, setApiKey] = useState("");

  useEffect(() => {
    const load = async () => {
      try {
        const p = await window.go.main.ConfigService.GetAvailableProviders();
        setProviders(p);
        if (p.length > 0) setSelectedProvider(p[0].id);
      } catch { /* noop */ }
    };
    load();
  }, []);

  const quickSetup = async () => {
    try {
      await window.go.main.ConfigService.QuickSetupProvider(selectedProvider, apiKey);
      showToast("Provider configured successfully!", "success");
      setApiKey("");
    } catch (e: any) {
      showToast(`Setup failed: ${e}`, "error");
    }
  };

  const selected = providers.find((p) => p.id === selectedProvider);

  return (
    <div className="space-y-6 max-w-3xl">
      <h2 className="text-2xl font-bold uppercase tracking-[0.2em] text-glow-pink">Configuration</h2>

      <NeonCard title="Quick Setup" collapsible variant="pink">
        <div className="space-y-3">
          <p className="text-sm text-white/40">
            Quickly configure an AI provider with one click.
          </p>
          <SearchableSelect
            label="Provider"
            placeholder="Search by provider or model name..."
            options={providers.map((p) => ({
              id: p.id,
              label: `${p.name} — ${p.modelName}`,
              sublabel: p.model,
              badge: p.needsKey ? undefined : "Local",
            }))}
            value={selectedProvider}
            onChange={(id) => { setSelectedProvider(id); setApiKey(""); }}
          />
          {selected?.needsKey && (
            <div>
              <NeonInput
                label="API Key"
                value={apiKey}
                onChange={setApiKey}
                type="password"
                placeholder="Enter your API key"
              />
              {selected?.keyURL && (
                <a
                  href={selected.keyURL}
                  target="_blank"
                  rel="noopener"
                  className="text-xs text-neon-cyan hover:underline mt-1 inline-block"
                >
                  Get API key →
                </a>
              )}
            </div>
          )}
          <NeonButton onClick={quickSetup} disabled={selected?.needsKey && !apiKey}>
            Configure {selected?.name ?? "Provider"}
          </NeonButton>
        </div>
      </NeonCard>

      <NeonCard title="Models" collapsible variant="green">
        <ProvidersSection showToast={showToast} />
      </NeonCard>

      <NeonCard title="Channels" collapsible variant="cyan">
        <ChannelsSection showToast={showToast} />
      </NeonCard>

      <NeonCard title="Agent Defaults" collapsible variant="purple">
        <AgentSection showToast={showToast} />
      </NeonCard>
    </div>
  );
}
