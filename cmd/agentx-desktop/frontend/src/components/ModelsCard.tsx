import type { ModelInfo } from "../lib/types";

interface Props {
  models: ModelInfo[];
}

export default function ModelsCard({ models }: Props) {
  const configured = models.filter((m) => m.hasKey);

  return (
    <div className="glass-card card-green p-5">
      <h3 className="text-xs font-bold uppercase tracking-widest text-white/70 mb-4">
        Models <span className="text-neon-green">({configured.length} configured)</span>
      </h3>
      <div className="space-y-2.5">
        {models.map((m) => (
          <div
            key={m.modelName}
            className="flex items-center justify-between text-sm"
          >
            <div>
              <span className={m.hasKey ? "text-white" : "text-white/25"}>
                {m.modelName}
              </span>
              <span className="text-white/15 text-xs ml-2 font-mono">{m.model}</span>
            </div>
            <span
              className={`w-2.5 h-2.5 rounded-full ${
                m.hasKey ? "bg-neon-green shadow-glow-green-sm" : "bg-white/15"
              }`}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
