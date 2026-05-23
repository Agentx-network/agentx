import { useState, useEffect } from "react";
import type { ProviderOption, DiscoveredModel } from "../lib/types";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import NeonInput from "../components/ui/NeonInput";
import SearchableSelect from "../components/ui/SearchableSelect";
import agentHero from "../assets/agent-hero.gif";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
  onComplete: () => void;
}

// Providers whose live model list we can fetch (GET /models).
const DISCOVERY_PROVIDERS = ["gemini", "google", "openai", "openrouter"];

export default function OnboardPage({ showToast, onComplete }: Props) {
  const [providers, setProviders] = useState<ProviderOption[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [saving, setSaving] = useState(false);

  // Dynamic model discovery + custom model entry.
  const [models, setModels] = useState<DiscoveredModel[]>([]);
  const [chosenModel, setChosenModel] = useState<string | null>(null);
  const [customModel, setCustomModel] = useState("");
  const [fetchingModels, setFetchingModels] = useState(false);
  const [showModelOptions, setShowModelOptions] = useState(false);

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
  const providerPrefix = selectedProvider ? selectedProvider.model.split("/")[0] : "";
  const canDiscover = DISCOVERY_PROVIDERS.includes(providerPrefix);

  // Reset model state whenever the provider changes.
  const onProviderChange = (id: string) => {
    setSelected(id);
    setApiKey("");
    setModels([]);
    setChosenModel(null);
    setCustomModel("");
    setShowModelOptions(false);
  };

  const fetchModels = async () => {
    if (!selectedProvider) return;
    setFetchingModels(true);
    try {
      const live = await window.go.main.ConfigService.ListProviderModels(
        providerPrefix, selectedProvider.apiBase, apiKey,
      );
      setModels(live);
      if (live.length === 0) {
        showToast("No models returned — you can still type a custom model below.", "error");
      } else {
        showToast(`Found ${live.length} models.`, "success");
      }
    } catch (e: any) {
      showToast(`Couldn't fetch models: ${e}. Use the default or type a custom model.`, "error");
    } finally {
      setFetchingModels(false);
    }
  };

  const handleSetup = async () => {
    if (!selectedProvider) return;
    setSaving(true);
    try {
      // Validate the key against the provider BEFORE saving, so a bad key
      // (typo, whitespace, wrong account) is caught here instead of at first
      // chat. Providers without a stable validation endpoint return "" (skip).
      if (selectedProvider.needsKey && apiKey.trim() !== "") {
        const reason = await window.go.main.ConfigService.ValidateProviderKey(
          providerPrefix, selectedProvider.apiBase, apiKey,
        );
        if (reason) {
          // The generic humanizer says 'check it in Config → Provider', but
          // the user is already on the provider screen at onboarding — so we
          // restate it in plain, direct language pointing to the field above.
          const lower = reason.toLowerCase();
          let friendly = reason;
          if (lower.includes("rejected the api key") || lower.includes("invalid")) {
            friendly = `Invalid ${selectedProvider.name} API key. Please double-check the key above and try again.`;
          } else if (lower.includes("couldn't reach") || lower.includes("network")) {
            friendly = `Couldn't reach ${selectedProvider.name} to verify the key — check your internet and try again.`;
          } else if (lower.includes("rate") || lower.includes("limit") || lower.includes("quota")) {
            friendly = `${selectedProvider.name} is rate-limiting key checks right now. Wait a moment and try again.`;
          }
          showToast(friendly, "error");
          setSaving(false);
          return;
        }
      }

      const custom = customModel.trim();
      const isCustomOrDynamic = custom !== "" || (chosenModel && chosenModel !== selectedProvider.model);
      if (selectedProvider.needsKey && isCustomOrDynamic) {
        const ref = custom || chosenModel!;
        const label = custom || models.find((m) => m.id === chosenModel)?.label || ref;
        await window.go.main.ConfigService.SetupModel(label, ref, selectedProvider.apiBase, apiKey);
      } else {
        // Default catalog model (or local/no-key provider).
        await window.go.main.ConfigService.QuickSetupProvider(selected!, apiKey);
      }
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
      <div className="text-center space-y-4">
        <img src={agentHero} alt="" className="w-20 h-20 mx-auto rounded-full border-2 border-neon-pink/25 shadow-[0_0_30px_rgba(255,0,128,0.2)]" />
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
            onChange={onProviderChange}
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

                {/* The provider dropdown already sets a sensible default model
                    (shown above), so model selection is collapsed by default.
                    Expand only to pull the provider's latest live list or type
                    a custom model id — the few cases the catalog can't cover. */}
                {!showModelOptions ? (
                  <button
                    onClick={() => setShowModelOptions(true)}
                    className="text-xs text-white/35 hover:text-neon-cyan transition-colors"
                  >
                    ▸ Change model (use latest or custom)
                  </button>
                ) : (
                  <div className="space-y-2 pt-2 border-t border-white/10">
                    <div className="flex items-center justify-between">
                      <span className="text-xs uppercase tracking-widest text-white/50">Model</span>
                      {canDiscover && (
                        <button
                          onClick={fetchModels}
                          disabled={!apiKey || fetchingModels}
                          className="text-xs text-neon-cyan hover:text-glow-cyan disabled:text-white/20 uppercase tracking-widest"
                        >
                          {fetchingModels ? "Fetching…" : "↻ Fetch latest models"}
                        </button>
                      )}
                    </div>

                    {models.length > 0 ? (
                      <SearchableSelect
                        label=""
                        placeholder="Pick a model…"
                        options={models.map((m) => ({ id: m.id, label: m.label, sublabel: m.id }))}
                        value={chosenModel ?? selectedProvider.model}
                        onChange={(id) => { setChosenModel(id); setCustomModel(""); }}
                      />
                    ) : (
                      <p className="text-xs text-white/35">
                        Default: <span className="font-mono text-white/60">{selectedProvider.model}</span>
                        {canDiscover ? " — or fetch the latest list above." : ""}
                      </p>
                    )}

                    <NeonInput
                      label="Custom model (optional)"
                      value={customModel}
                      onChange={setCustomModel}
                      placeholder={`e.g. ${providerPrefix}/your-model-id`}
                    />
                  </div>
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
