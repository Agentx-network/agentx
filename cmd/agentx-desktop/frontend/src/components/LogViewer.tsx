import { useState, useEffect, useRef } from "react";

interface Props {
  onRefresh?: () => void;
}

export default function LogViewer({ onRefresh }: Props) {
  const [logs, setLogs] = useState("");
  const [loading, setLoading] = useState(false);
  const ref = useRef<HTMLPreElement>(null);

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const text = await window.go.main.DashboardService.GetLogs(100);
      setLogs(text);
      if (ref.current) {
        ref.current.scrollTop = ref.current.scrollHeight;
      }
    } catch {
      setLogs("No logs available");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
    const interval = setInterval(fetchLogs, 10000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="glass-card p-5">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-white/70">Logs</h3>
        <button
          onClick={() => { fetchLogs(); onRefresh?.(); }}
          disabled={loading}
          className="text-xs text-neon-cyan hover:text-neon-cyan/80 transition-colors"
        >
          {loading ? "Loading..." : "Refresh"}
        </button>
      </div>
      <pre
        ref={ref}
        className="bg-black/40 rounded-lg p-3 text-xs text-white/70 font-mono overflow-auto max-h-64 whitespace-pre-wrap"
      >
        {logs || "No logs available"}
      </pre>
    </div>
  );
}
