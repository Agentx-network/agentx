import { useState, useEffect } from "react";

/**
 * Returns the number of items that should be visible so far (0 → total).
 * Each item appears `intervalMs` after the previous one.
 */
export function useStaggeredEntrance(total: number, intervalMs = 100): number {
  const [visible, setVisible] = useState(0);

  useEffect(() => {
    if (visible >= total) return;
    const id = setTimeout(() => setVisible((v) => v + 1), intervalMs);
    return () => clearTimeout(id);
  }, [visible, total, intervalMs]);

  // Reset when total changes (e.g. page re-mount)
  useEffect(() => {
    setVisible(0);
  }, [total]);

  return visible;
}
