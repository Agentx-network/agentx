package tools

import (
	"context"
	"strings"
	"testing"
)

// CancelAllJobs is the escape hatch behind the user-typed "/stop" — it must
// drop every persisted job (so a runaway reminder/command stops immediately)
// and return the count it removed.
func TestCancelAllJobs(t *testing.T) {
	tool := newCronToolForGuardTests(t)
	// Desktop is a valid local delivery target (no push channel needed), so
	// reminders queued here don't get rejected by resolveReminderTarget.
	tool.SetContext("desktop", "chat")
	ctx := context.Background()

	// No jobs yet — count is 0, nothing crashes.
	if n := tool.CancelAllJobs(); n != 0 {
		t.Errorf("with no jobs, CancelAllJobs() = %d, want 0", n)
	}

	// Add three real reminder jobs.
	for i, msg := range []string{"first", "second", "third"} {
		res := tool.Execute(ctx, map[string]any{
			"action":     "add",
			"message":    msg,
			"at_seconds": float64(60 + i*60),
		})
		if res.IsError {
			t.Fatalf("setup: failed to add job %d: %s", i, res.ForLLM)
		}
	}

	// CancelAllJobs reports the count it cleared.
	if n := tool.CancelAllJobs(); n != 3 {
		t.Errorf("CancelAllJobs() = %d, want 3", n)
	}
	// And subsequent calls find no work.
	if n := tool.CancelAllJobs(); n != 0 {
		t.Errorf("after cancel-all, CancelAllJobs() = %d, want 0", n)
	}

	// Sanity: listJobs through the cron tool should now show none.
	res := tool.Execute(ctx, map[string]any{"action": "list"})
	if res.IsError {
		t.Fatalf("list failed: %s", res.ForLLM)
	}
	if !strings.Contains(res.ForLLM, "No") && res.ForLLM != "" && !strings.Contains(res.ForLLM, "0") {
		// Some implementations return "No jobs" or an empty list summary — both fine.
		// We only fail if the list claims jobs still exist.
		if strings.Contains(strings.ToLower(res.ForLLM), "first") || strings.Contains(strings.ToLower(res.ForLLM), "second") {
			t.Errorf("list still shows cancelled jobs: %q", res.ForLLM)
		}
	}
}

// CancelAllJobs must not panic on a CronTool with no service wired (defensive,
// since the helper is reachable from the /stop command handler).
func TestCancelAllJobs_NilServiceSafe(t *testing.T) {
	tool := &CronTool{}
	if n := tool.CancelAllJobs(); n != 0 {
		t.Errorf("CancelAllJobs on empty tool should return 0, got %d", n)
	}
}
