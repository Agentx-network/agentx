import type { ModelInfo } from "../lib/types";

interface Props {
  models: ModelInfo[];
}

export default function ModelsCard({ models }: Props) {
  const configured = models.filter((m) => m.hasKey);

  return (
    <div className="glass-card p-5">
      <h3 className="text-sm font-medium text-white/70 mb-4">
        Models ({configured.length} configured)
      </h3>
      <div className="space-y-2">
        {models.map((m) => (
          <div
            key={m.modelName}
            className="flex items-center justify-between text-sm"
          >
            <div>
              <span className={m.hasKey ? "text-white" : "text-white/30"}>
                {m.modelName}
              </span>
              <span className="text-white/20 text-xs ml-2">{m.model}</span>
            </div>
            <span
              className={`w-2 h-2 rounded-full ${
                m.hasKey ? "bg-neon-pink" : "bg-white/20"
              }`}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
