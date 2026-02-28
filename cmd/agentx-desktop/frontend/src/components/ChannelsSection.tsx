import { useState, useEffect } from "react";
import type { ChannelInfo } from "../lib/types";
import NeonToggle from "./ui/NeonToggle";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

const channelKeys: Record<string, string> = {
  Telegram: "telegram",
  Discord: "discord",
  Slack: "slack",
  WhatsApp: "whatsapp",
  Feishu: "feishu",
  DingTalk: "dingtalk",
  QQ: "qq",
  LINE: "line",
  OneBot: "onebot",
  WeCom: "wecom",
  WeComApp: "wecom_app",
  MaixCam: "maixcam",
};

export default function ChannelsSection({ showToast }: Props) {
  const [channels, setChannels] = useState<ChannelInfo[]>([]);

  const load = async () => {
    try {
      const status = await window.go.main.DashboardService.GetStatus();
      setChannels(status.channels ?? []);
    } catch { /* noop */ }
  };

  useEffect(() => { load(); }, []);

  const toggle = async (name: string, enabled: boolean) => {
    const key = channelKeys[name];
    if (!key) return;
    try {
      await window.go.main.ConfigService.SetChannelEnabled(key, enabled);
      showToast(`${name} ${enabled ? "enabled" : "disabled"}`, "success");
      load();
    } catch (e: any) {
      showToast(e.toString(), "error");
    }
  };

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-white/70">Channels</h3>
      <div className="space-y-2">
        {channels.map((ch) => (
          <div key={ch.name} className="flex items-center justify-between py-1.5">
            <span className="text-sm text-white/80">{ch.name}</span>
            <NeonToggle
              checked={ch.enabled}
              onChange={(v) => toggle(ch.name, v)}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
