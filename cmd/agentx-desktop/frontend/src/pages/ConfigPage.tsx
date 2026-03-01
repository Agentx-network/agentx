import React, { useState, useEffect } from "react";
import type { ProviderOption, ModelConfig, ChannelInfo, AgentDefaults } from "../lib/types";
import NeonButton from "../components/ui/NeonButton";
import NeonInput from "../components/ui/NeonInput";
import SearchableSelect from "../components/ui/SearchableSelect";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

type Tab = "provider" | "channels" | "agent";

export default function ConfigPage({ showToast }: Props) {
  const [tab, setTab] = useState<Tab>("provider");
  const [providers, setProviders] = useState<ProviderOption[]>([]);
  const [models, setModels] = useState<ModelConfig[]>([]);
  const [channels, setChannels] = useState<ChannelInfo[]>([]);
  const [defaults, setDefaults] = useState<AgentDefaults | null>(null);

  useEffect(() => {
    loadAll();
  }, []);

  const loadAll = async () => {
    try {
      const [p, m, status, d] = await Promise.all([
        window.go.main.ConfigService.GetAvailableProviders(),
        window.go.main.ConfigService.GetModelList(),
        window.go.main.DashboardService.GetStatus(),
        window.go.main.ConfigService.GetAgentDefaults(),
      ]);
      setProviders(p);
      setModels(m ?? []);
      setChannels(status.channels ?? []);
      setDefaults(d);
    } catch { /* noop */ }
  };

  const tabs: { key: Tab; label: string; count?: number }[] = [
    { key: "provider", label: "Provider", count: models.filter(m => m.api_key).length },
    { key: "channels", label: "Channels", count: channels.filter(c => c.enabled).length },
    { key: "agent", label: "Agent" },
  ];

  return (
    <div className="max-w-2xl space-y-5">
      <h2 className="text-2xl font-bold uppercase tracking-[0.2em] text-glow-pink">
        Config
      </h2>

      {/* Active config summary bar */}
      <div className="flex gap-3">
        <StatusChip
          label="Model"
          value={defaults?.model_name || defaults?.model || "Not set"}
          active={!!defaults?.model_name || !!defaults?.model}
        />
        <StatusChip
          label="Channels"
          value={`${channels.filter(c => c.enabled).length} active`}
          active={channels.some(c => c.enabled)}
        />
        <StatusChip
          label="Providers"
          value={`${models.filter(m => m.api_key).length} configured`}
          active={models.some(m => m.api_key)}
        />
      </div>

      {/* Tab bar */}
      <div className="flex gap-1 bg-white/[0.03] rounded-lg p-1">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex-1 px-4 py-2 rounded-md text-xs font-bold uppercase tracking-widest transition-all ${
              tab === t.key
                ? "bg-neon-pink/20 text-neon-pink shadow-[0_0_12px_rgba(255,0,102,0.15)]"
                : "text-white/40 hover:text-white/60 hover:bg-white/[0.03]"
            }`}
          >
            {t.label}
            {t.count !== undefined && (
              <span className={`ml-1.5 text-[10px] ${tab === t.key ? "text-neon-pink/60" : "text-white/20"}`}>
                ({t.count})
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {tab === "provider" && (
        <ProviderTab
          providers={providers}
          models={models}
          showToast={showToast}
          onRefresh={loadAll}
        />
      )}
      {tab === "channels" && (
        <ChannelsTab
          channels={channels}
          showToast={showToast}
          onRefresh={loadAll}
        />
      )}
      {tab === "agent" && (
        <AgentTab
          defaults={defaults}
          setDefaults={setDefaults}
          showToast={showToast}
        />
      )}
    </div>
  );
}

/* ─── Status chip ─── */
function StatusChip({ label, value, active }: { label: string; value: string; active: boolean }) {
  return (
    <div className="flex-1 bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2">
      <div className="text-[10px] uppercase tracking-widest text-white/30 mb-0.5">{label}</div>
      <div className="flex items-center gap-1.5">
        <div className={`w-1.5 h-1.5 rounded-full ${active ? "bg-neon-green shadow-[0_0_6px_rgba(0,255,65,0.5)]" : "bg-white/15"}`} />
        <span className="text-xs text-white/70 truncate">{value}</span>
      </div>
    </div>
  );
}

/* ─── Provider Tab ─── */
function ProviderTab({
  providers,
  models,
  showToast,
  onRefresh,
}: {
  providers: ProviderOption[];
  models: ModelConfig[];
  showToast: Props["showToast"];
  onRefresh: () => void;
}) {
  // Default to the provider that has a key configured, or first available
  const activeModel = models.find(m => m.api_key) || models[0];
  const defaultId = activeModel
    ? providers.find(p => p.model === activeModel.model)?.id ?? ""
    : "";
  const [selectedProvider, setSelectedProvider] = useState("");

  useEffect(() => {
    if (!selectedProvider && providers.length > 0) {
      setSelectedProvider(defaultId || providers[0].id);
    }
  }, [providers, defaultId, selectedProvider]);

  // Find matching configured model for the selected provider
  const selected = providers.find((p) => p.id === selectedProvider);
  const configuredModel = models.find(m => m.model === selected?.model);

  const save = async () => {
    try {
      await window.go.main.ConfigService.QuickSetupProvider(selectedProvider, apiKey);
      showToast("Provider configured!", "success");
      setApiKey("");
      onRefresh();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  const [apiKey, setApiKey] = useState("");

  const maskKey = (key: string) => {
    if (!key) return "";
    if (key.length <= 8) return "********";
    return key.slice(0, 4) + "****" + key.slice(-4);
  };

  // Build options with "Configured" badge for ones that have keys
  const configuredModels = new Set(models.filter(m => m.api_key).map(m => m.model));

  return (
    <div className="glass-card p-4 space-y-4">
      <SearchableSelect
        label="Provider"
        placeholder="Search provider or model..."
        options={providers.map((p) => ({
          id: p.id,
          label: `${p.name} — ${p.modelName}`,
          sublabel: p.model,
          badge: configuredModels.has(p.model) ? "Active" : p.needsKey ? undefined : "Local",
        }))}
        value={selectedProvider}
        onChange={(id) => { setSelectedProvider(id); setApiKey(""); }}
      />

      {/* Show current key status */}
      {configuredModel?.api_key && (
        <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-neon-green/5 border border-neon-green/15">
          <div className="w-1.5 h-1.5 rounded-full bg-neon-green shadow-[0_0_6px_rgba(0,255,65,0.5)]" />
          <span className="text-xs text-neon-green/80">Key set:</span>
          <span className="text-xs text-white/40 font-mono">{maskKey(configuredModel.api_key)}</span>
        </div>
      )}

      {selected?.needsKey && (
        <div>
          <NeonInput
            label={configuredModel?.api_key ? "Update API Key" : "API Key"}
            value={apiKey}
            onChange={setApiKey}
            type="password"
            placeholder={configuredModel?.api_key ? "Enter new key to update" : "Enter your API key"}
          />
          {selected?.keyURL && (
            <a href={selected.keyURL} target="_blank" rel="noopener" className="text-[11px] text-neon-cyan hover:underline mt-1 inline-block">
              Get API key →
            </a>
          )}
        </div>
      )}

      <NeonButton onClick={save} size="sm" disabled={selected?.needsKey && !apiKey}>
        {configuredModel?.api_key ? "Update" : "Configure"}
      </NeonButton>
    </div>
  );
}

/* ─── Channels Tab ─── */
const channelOptions = [
  { id: "telegram", name: "Telegram", placeholder: "123456:ABC-DEF...", helpUrl: "https://t.me/BotFather", helpText: "Get token from @BotFather" },
  { id: "discord", name: "Discord", placeholder: "MTIz...", helpUrl: "https://discord.com/developers/applications", helpText: "Bot token from Developer Portal" },
  { id: "slack", name: "Slack", placeholder: "xoxb-...", helpUrl: "https://api.slack.com/apps", helpText: "Bot token from Slack API" },
  { id: "whatsapp", name: "WhatsApp", placeholder: "Token...", helpUrl: "", helpText: "WhatsApp Business API token" },
  { id: "feishu", name: "Feishu", placeholder: "Token...", helpUrl: "", helpText: "Feishu bot token" },
  { id: "dingtalk", name: "DingTalk", placeholder: "Token...", helpUrl: "", helpText: "DingTalk bot token" },
  { id: "line", name: "LINE", placeholder: "Token...", helpUrl: "", helpText: "LINE channel access token" },
];

function ChannelsTab({
  channels,
  showToast,
  onRefresh,
}: {
  channels: ChannelInfo[];
  showToast: Props["showToast"];
  onRefresh: () => void;
}) {
  const enabledChannels = channels.filter(c => c.enabled);
  // Default to first enabled channel
  const defaultChannel = enabledChannels.length > 0
    ? channelOptions.find(co => co.name === enabledChannels[0].name)?.id ?? null
    : null;
  const [selected, setSelected] = useState<string | null>(defaultChannel);
  const [token, setToken] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!selected && defaultChannel) {
      setSelected(defaultChannel);
    }
  }, [defaultChannel, selected]);

  const selectedChannel = channelOptions.find((c) => c.id === selected);
  const isEnabled = selectedChannel ? enabledChannels.some(c => c.name === selectedChannel.name) : false;

  const handleSetup = async () => {
    if (!selected || !token) return;
    setSaving(true);
    try {
      await window.go.main.ConfigService.QuickSetupChannel(selected, token);
      showToast(`${selectedChannel?.name} configured!`, "success");
      setToken("");
      onRefresh();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      {/* Active channels */}
      {enabledChannels.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {enabledChannels.map((ch) => (
            <span key={ch.name} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-neon-green/5 border border-neon-green/15 text-xs text-neon-green/80">
              <div className="w-1.5 h-1.5 rounded-full bg-neon-green shadow-[0_0_6px_rgba(0,255,65,0.5)]" />
              {ch.name}
            </span>
          ))}
        </div>
      )}

      <div className="glass-card p-4 space-y-4">
        <SearchableSelect
          label="Channel"
          placeholder="Search channels..."
          options={channelOptions.map((ch) => ({
            id: ch.id,
            label: ch.name,
            sublabel: ch.helpText,
            badge: enabledChannels.some(c => c.name === ch.name) ? "Active" : undefined,
          }))}
          value={selected}
          onChange={(id) => { setSelected(id); setToken(""); }}
        />

        {selectedChannel && (
          <>
            {isEnabled && (
              <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-neon-green/5 border border-neon-green/15">
                <div className="w-1.5 h-1.5 rounded-full bg-neon-green shadow-[0_0_6px_rgba(0,255,65,0.5)]" />
                <span className="text-xs text-neon-green/80">{selectedChannel.name} is active</span>
              </div>
            )}
            <NeonInput
              label={isEnabled ? "Update Bot Token" : "Bot Token"}
              value={token}
              onChange={setToken}
              type="password"
              placeholder={isEnabled ? "Enter new token to update" : selectedChannel.placeholder}
            />
            {selectedChannel.helpUrl && (
              <a href={selectedChannel.helpUrl} target="_blank" rel="noopener" className="text-[11px] text-neon-cyan hover:underline inline-block">
                {selectedChannel.helpText} →
              </a>
            )}
            <NeonButton onClick={handleSetup} size="sm" disabled={!token || saving}>
              {saving ? "Saving..." : isEnabled ? "Update" : "Configure"}
            </NeonButton>
          </>
        )}
      </div>
    </div>
  );
}

/* ─── Agent Tab ─── */
interface BootstrapFile {
  name: string;
  path: string;
  content: string;
  exists: boolean;
}

interface SkillEntry {
  name: string;
  source: string;
  description: string;
  path: string;
}

const fileDescriptions: Record<string, string> = {
  "SOUL.md": "Personality, tone, and values",
  "IDENTITY.md": "Name, purpose, and capabilities",
  "AGENTS.md": "Workflow rules and multi-channel behavior",
  "USER.md": "Your profile and preferences",
};

function AgentTab({
  defaults,
  setDefaults,
  showToast,
}: {
  defaults: AgentDefaults | null;
  setDefaults: (d: AgentDefaults) => void;
  showToast: Props["showToast"];
}) {
  const [subTab, setSubTab] = useState<"settings" | "files" | "skills">("settings");
  const [files, setFiles] = useState<BootstrapFile[]>([]);
  const [editingFile, setEditingFile] = useState<string | null>(null);
  const [editContent, setEditContent] = useState("");
  const [saving, setSaving] = useState(false);
  const [skills, setSkills] = useState<SkillEntry[]>([]);
  const [installRepo, setInstallRepo] = useState("");
  const [installing, setInstalling] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<{ slug: string; displayName: string; summary: string; version: string; registry: string }[]>([]);
  const [searching, setSearching] = useState(false);
  const [installingSlug, setInstallingSlug] = useState<string | null>(null);

  useEffect(() => {
    loadFiles();
    loadSkills();
  }, []);

  const loadFiles = async () => {
    try {
      const f = await window.go.main.AgentSetupService.GetBootstrapFiles();
      setFiles(f);
    } catch { /* noop */ }
  };

  const loadSkills = async () => {
    try {
      const s = await window.go.main.AgentSetupService.ListSkills();
      setSkills(s || []);
    } catch { /* noop */ }
  };

  const saveDefaults = async () => {
    if (!defaults) return;
    try {
      await window.go.main.ConfigService.UpdateAgentDefaults(defaults);
      showToast("Saved", "success");
    } catch (e: any) {
      showToast(e.toString(), "error");
    }
  };

  const handleCreateDefaults = async () => {
    try {
      await window.go.main.AgentSetupService.CreateDefaultBootstrapFiles();
      showToast("Default files created!", "success");
      loadFiles();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  const handleEditFile = (file: BootstrapFile) => {
    setEditingFile(file.name);
    setEditContent(file.content);
  };

  const handleSaveFile = async () => {
    if (!editingFile) return;
    setSaving(true);
    try {
      await window.go.main.AgentSetupService.SaveBootstrapFile(editingFile, editContent);
      showToast(`${editingFile} saved!`, "success");
      setEditingFile(null);
      loadFiles();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setSaving(false);
    }
  };

  const handleInstallSkill = async () => {
    if (!installRepo.trim()) return;
    setInstalling(true);
    try {
      await window.go.main.AgentSetupService.InstallSkill(installRepo.trim());
      showToast("Skill installed!", "success");
      setInstallRepo("");
      loadSkills();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setInstalling(false);
    }
  };

  const handleSearch = async () => {
    if (!searchQuery.trim()) return;
    setSearching(true);
    try {
      const results = await window.go.main.AgentSetupService.SearchSkills(searchQuery.trim());
      setSearchResults(results || []);
    } catch (e: any) {
      showToast(`Search failed: ${e}`, "error");
      setSearchResults([]);
    } finally {
      setSearching(false);
    }
  };

  const handleSearchKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") handleSearch();
  };

  const handleInstallFromRegistry = async (slug: string) => {
    setInstallingSlug(slug);
    try {
      await window.go.main.AgentSetupService.InstallFromRegistry(slug);
      showToast(`${slug} installed!`, "success");
      loadSkills();
      // Remove from search results
      setSearchResults((prev) => prev.filter((r) => r.slug !== slug));
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setInstallingSlug(null);
    }
  };

  const handleInstallBuiltins = async () => {
    setInstalling(true);
    try {
      await window.go.main.AgentSetupService.InstallBuiltinSkills();
      showToast("Builtin skills installed!", "success");
      loadSkills();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setInstalling(false);
    }
  };

  const handleRemoveSkill = async (name: string) => {
    try {
      await window.go.main.AgentSetupService.RemoveSkill(name);
      showToast(`${name} removed`, "success");
      loadSkills();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  const hasAnyFiles = files.some((f) => f.exists);

  return (
    <div className="space-y-4">
      {/* Sub-tabs */}
      <div className="flex gap-1 bg-white/[0.02] rounded-md p-0.5">
        {(["settings", "files", "skills"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setSubTab(t)}
            className={`flex-1 px-3 py-1.5 rounded text-[10px] font-bold uppercase tracking-widest transition-all ${
              subTab === t
                ? "bg-white/[0.06] text-white/80"
                : "text-white/25 hover:text-white/40"
            }`}
          >
            {t === "files" ? "Identity" : t === "skills" ? `Skills (${skills.length})` : "Settings"}
          </button>
        ))}
      </div>

      {/* Settings sub-tab */}
      {subTab === "settings" && defaults && (
        <div className="glass-card p-4 space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <NeonInput label="Model" value={defaults.model_name ?? defaults.model ?? ""} onChange={(v) => setDefaults({ ...defaults, model_name: v })} />
            <NeonInput label="Max Tokens" value={String(defaults.max_tokens)} onChange={(v) => setDefaults({ ...defaults, max_tokens: parseInt(v) || 0 })} />
            <NeonInput label="Temperature" value={defaults.temperature !== undefined ? String(defaults.temperature) : ""} onChange={(v) => setDefaults({ ...defaults, temperature: v ? parseFloat(v) : undefined })} placeholder="auto" />
            <NeonInput label="Max Tool Iterations" value={String(defaults.max_tool_iterations)} onChange={(v) => setDefaults({ ...defaults, max_tool_iterations: parseInt(v) || 0 })} />
          </div>
          <NeonInput label="Workspace" value={defaults.workspace} onChange={(v) => setDefaults({ ...defaults, workspace: v })} />
          <NeonButton onClick={saveDefaults} size="sm">Save</NeonButton>
        </div>
      )}

      {/* Identity files sub-tab */}
      {subTab === "files" && (
        <div className="space-y-3">
          {!hasAnyFiles && (
            <div className="glass-card p-4 text-center space-y-3">
              <p className="text-sm text-white/40">No identity files yet.</p>
              <NeonButton onClick={handleCreateDefaults} size="sm" className="w-full">Create Defaults</NeonButton>
            </div>
          )}

          {editingFile ? (
            <div className="glass-card p-4 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold uppercase tracking-widest text-neon-pink">{editingFile}</span>
                <button onClick={() => setEditingFile(null)} className="text-xs text-white/30 hover:text-white/60">Cancel</button>
              </div>
              <textarea
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                rows={14}
                className="w-full bg-black/40 border-2 border-neon-purple/20 rounded-lg px-4 py-3 text-sm text-white/80 font-mono leading-relaxed focus:outline-none focus:border-neon-pink/40 transition-all resize-y"
              />
              <NeonButton onClick={handleSaveFile} disabled={saving} size="sm" className="w-full">
                {saving ? "Saving..." : "Save"}
              </NeonButton>
            </div>
          ) : (
            <div className="glass-card divide-y divide-white/[0.04]">
              {files.map((file) => (
                <div key={file.name} className="flex items-center justify-between px-4 py-3">
                  <div>
                    <span className="text-sm text-white/80 font-medium">{file.name}</span>
                    <span className="text-[11px] text-white/25 ml-2">{fileDescriptions[file.name] || ""}</span>
                  </div>
                  {file.exists ? (
                    <button onClick={() => handleEditFile(file)} className="text-xs text-neon-pink/50 hover:text-neon-pink transition-colors uppercase tracking-widest">Edit</button>
                  ) : (
                    <button onClick={() => { setEditingFile(file.name); setEditContent(`# ${file.name.replace(".md", "")}\n\n`); }} className="text-xs text-white/30 hover:text-white/50 transition-colors uppercase tracking-widest">Create</button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Skills sub-tab */}
      {subTab === "skills" && (
        <div className="space-y-3">
          {/* Search ClawHub */}
          <div className="glass-card p-4 space-y-3">
            <div className="text-[10px] uppercase tracking-widest text-white/25 mb-1">Search ClawHub Registry</div>
            <div className="flex gap-2">
              <div className="flex-1">
                <NeonInput
                  value={searchQuery}
                  onChange={setSearchQuery}
                  onKeyDown={handleSearchKeyDown}
                  placeholder="Search skills... (e.g. web search, docker)"
                />
              </div>
              <NeonButton onClick={handleSearch} disabled={!searchQuery.trim() || searching} size="sm">
                {searching ? "..." : "Search"}
              </NeonButton>
            </div>

            {/* Search results */}
            {searchResults.length > 0 && (
              <div className="space-y-2 mt-2">
                {searchResults.map((r) => {
                  const alreadyInstalled = skills.some(s => s.name === r.slug);
                  return (
                    <div key={r.slug} className="flex items-center gap-3 px-3 py-2.5 rounded-lg bg-white/[0.02] border border-white/[0.06] hover:border-neon-cyan/20 transition-colors">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm text-white/80 font-medium">{r.displayName || r.slug}</span>
                          {r.version && <span className="text-[10px] text-white/20 font-mono">v{r.version}</span>}
                        </div>
                        {r.summary && <p className="text-[11px] text-white/30 mt-0.5 truncate">{r.summary}</p>}
                      </div>
                      {alreadyInstalled ? (
                        <span className="text-[10px] text-neon-green/60 uppercase tracking-widest shrink-0">Installed</span>
                      ) : (
                        <NeonButton
                          onClick={() => handleInstallFromRegistry(r.slug)}
                          disabled={installingSlug === r.slug}
                          size="sm"
                        >
                          {installingSlug === r.slug ? "..." : "Install"}
                        </NeonButton>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
            {searchResults.length === 0 && searchQuery && !searching && (
              <p className="text-[11px] text-white/20 text-center py-2">No results. Try a different query.</p>
            )}
          </div>

          {/* Builtins + GitHub install */}
          <div className="glass-card p-4 space-y-3">
            <NeonButton onClick={handleInstallBuiltins} disabled={installing} variant="ghost" size="sm" className="w-full">
              {installing ? "Installing..." : "Install Builtin Skills (web-search, calculator, summarizer, code-runner)"}
            </NeonButton>
            <div className="border-t border-white/[0.04] pt-3">
              <div className="text-[10px] uppercase tracking-widest text-white/25 mb-2">Install from GitHub</div>
              <div className="flex gap-2">
                <div className="flex-1">
                  <NeonInput value={installRepo} onChange={setInstallRepo} placeholder="owner/repo" />
                </div>
                <NeonButton onClick={handleInstallSkill} disabled={!installRepo.trim() || installing} size="sm">
                  {installing ? "..." : "Install"}
                </NeonButton>
              </div>
            </div>
          </div>

          {/* Installed skills */}
          {skills.length > 0 && (
            <div className="glass-card overflow-hidden">
              <div className="px-4 py-2.5 border-b border-white/[0.04]">
                <span className="text-[10px] uppercase tracking-widest text-white/25">Installed ({skills.length})</span>
              </div>
              <div className="divide-y divide-white/[0.04]">
                {skills.map((skill) => (
                  <div key={skill.name} className="flex items-center justify-between px-4 py-3">
                    <div>
                      <span className="text-sm text-white/80 font-medium">{skill.name}</span>
                      <span className="text-[10px] text-neon-cyan/50 bg-neon-cyan/5 px-1.5 py-0.5 rounded ml-2 uppercase tracking-widest">{skill.source}</span>
                      {skill.description && <p className="text-[11px] text-white/25 mt-0.5">{skill.description}</p>}
                    </div>
                    <button onClick={() => handleRemoveSkill(skill.name)} className="text-xs text-red-400/40 hover:text-red-400 transition-colors uppercase tracking-widest">Remove</button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
