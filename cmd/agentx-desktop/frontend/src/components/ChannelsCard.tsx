import type { ChannelInfo } from "../lib/types";

interface Props {
  channels: ChannelInfo[];
}

export default function ChannelsCard({ channels }: Props) {
  const active = channels.filter((c) => c.enabled);

  return (
    <div className="glass-card p-5">
      <h3 className="text-sm font-medium text-white/70 mb-4">
        Channels ({active.length} active)
      </h3>
      <div className="space-y-2">
        {channels.map((ch) => (
          <div
            key={ch.name}
            className="flex items-center justify-between text-sm"
          >
            <span className={ch.enabled ? "text-white" : "text-white/30"}>
              {ch.name}
            </span>
            <span
              className={`w-2 h-2 rounded-full ${
                ch.enabled ? "bg-neon-cyan" : "bg-white/20"
              }`}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
