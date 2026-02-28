import { useState } from "react";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import NeonInput from "../components/ui/NeonInput";
import SearchableSelect from "../components/ui/SearchableSelect";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
  onComplete: () => void;
}

const channels = [
  { id: "telegram", name: "Telegram", placeholder: "123456:ABC-DEF...", helpUrl: "https://t.me/BotFather", helpText: "Get token from @BotFather" },
  { id: "discord", name: "Discord", placeholder: "MTIz...", helpUrl: "https://discord.com/developers/applications", helpText: "Bot token from Developer Portal" },
  { id: "slack", name: "Slack", placeholder: "xoxb-...", helpUrl: "https://api.slack.com/apps", helpText: "Bot token from Slack API" },
];

export default function ChannelSetupPage({ showToast, onComplete }: Props) {
  const [selected, setSelected] = useState<string | null>(null);
  const [token, setToken] = useState("");
  const [saving, setSaving] = useState(false);

  const selectedChannel = channels.find((c) => c.id === selected);

  const handleSetup = async () => {
    if (!selected || !token) return;
    setSaving(true);
    try {
      await window.go.main.ConfigService.QuickSetupChannel(selected, token);
      showToast(`${selectedChannel?.name} configured!`, "success");
      onComplete();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="max-w-xl mx-auto space-y-6">
      <div className="text-center space-y-2">
        <h2 className="text-3xl font-bold uppercase tracking-[0.2em] text-glow-pink">Connect a Channel</h2>
        <p className="text-white/35 text-sm">
          Choose where your agent will receive messages.
        </p>
      </div>

      <NeonCard>
        <div className="space-y-4">
          <SearchableSelect
            label="Channel"
            placeholder="Search channels..."
            options={channels.map((ch) => ({
              id: ch.id,
              label: ch.name,
              sublabel: ch.helpText,
            }))}
            value={selected}
            onChange={(id) => { setSelected(id); setToken(""); }}
          />
        </div>
      </NeonCard>

      {selectedChannel && (
        <NeonCard variant="cyan" glow>
          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-widest text-white/90">{selectedChannel.name}</h3>
            <NeonInput
              label="Bot Token"
              value={token}
              onChange={setToken}
              type="password"
              placeholder={selectedChannel.placeholder}
            />
            <a
              href={selectedChannel.helpUrl}
              target="_blank"
              rel="noopener"
              className="text-xs text-neon-cyan hover:underline inline-block text-glow-cyan"
            >
              {selectedChannel.helpText} →
            </a>
            <NeonButton
              onClick={handleSetup}
              disabled={!token || saving}
              size="lg"
              className="w-full"
            >
              {saving ? "Saving..." : "Save & Continue →"}
            </NeonButton>
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
