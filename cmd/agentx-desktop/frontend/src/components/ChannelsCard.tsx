import type { ChannelInfo } from "../lib/types";

interface Props {
  channels: ChannelInfo[];
}

export default function ChannelsCard({ channels }: Props) {
  const active = channels.filter((c) => c.enabled);

  return (
    <div className="glass-card card-cyan p-5">
      <h3 className="text-xs font-bold uppercase tracking-widest text-white/70 mb-4">
        Channels <span className="text-neon-cyan">({active.length} active)</span>
      </h3>
      <div className="space-y-2.5">
        {channels.map((ch) => (
          <div
            key={ch.name}
            className="flex items-center justify-between text-sm"
          >
            <span className={ch.enabled ? "text-white" : "text-white/25"}>
              {ch.name}
            </span>
            <span
              className={`w-2.5 h-2.5 rounded-full ${
                ch.enabled ? "bg-neon-cyan shadow-[0_0_8px_rgba(0,255,255,0.5)]" : "bg-white/15"
              }`}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
