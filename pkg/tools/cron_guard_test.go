package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/cron"
)

// newCronToolForGuardTests builds a minimal CronTool whose cronService is real
// (so addJob can reach AddJob without nil-panicking) but writes to a temp dir.
func newCronToolForGuardTests(t *testing.T) *CronTool {
	t.Helper()
	store := filepath.Join(t.TempDir(), "jobs.json")
	return &CronTool{
		cfg:         &config.Config{},
		cronService: cron.NewCronService(store, nil),
	}
}

// Regression for the "spam every 10 seconds with 'Scheduled command X executed:
// (no output)'" bug. A weak model created a recurring shell-command cron with
// every_seconds=10 for a "connect Telegram"-style request; it ran forever after.
// addJob must reject recurring commands tighter than 60 seconds before any job
// is persisted.
func TestCronAddJob_RecurringCommandFloor(t *testing.T) {
	cases := []struct {
		name        string
		everySec    float64
		wantBlocked bool
	}{
		{"10s floor blocked", 10, true},
		{"30s floor blocked", 30, true},
		{"59s floor blocked", 59, true},
		{"60s allowed", 60, false}, // exactly at floor — passes the guard
		{"3600s allowed", 3600, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tool := newCronToolForGuardTests(t)
			res := tool.Execute(context.Background(), map[string]any{
				"action":        "add",
				"message":       "monitor disk",
				"command":       "df -h",
				"every_seconds": c.everySec,
			})
			blocked := res.IsError && strings.Contains(res.ForLLM, "every_seconds")
			if blocked != c.wantBlocked {
				t.Errorf("every_seconds=%v: blocked=%v want %v (ForLLM=%q)",
					c.everySec, blocked, c.wantBlocked, res.ForLLM)
			}
		})
	}
}

// A one-time scheduled command (at_seconds) has no recurrence and must NOT be
// rejected by the recurring-floor guard, even at small delays.
func TestCronAddJob_OneTimeCommandNotFloorBlocked(t *testing.T) {
	tool := newCronToolForGuardTests(t)
	res := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "one-off",
		"command":    "echo hi",
		"at_seconds": float64(5),
	})
	// at_seconds=5 is fine; the floor only applies to every_seconds.
	if res.IsError && strings.Contains(res.ForLLM, "every_seconds") {
		t.Errorf("one-time command must not be blocked by recurring floor; got: %q", res.ForLLM)
	}
}

// Recurring REMINDERS (no command) should be allowed at small intervals —
// the floor only restricts shell commands. (e.g. "ping me every 30 seconds"
// is unusual but valid.)
func TestCronAddJob_RecurringReminderUnaffected(t *testing.T) {
	tool := newCronToolForGuardTests(t)
	res := tool.Execute(context.Background(), map[string]any{
		"action":        "add",
		"message":       "ping",
		"every_seconds": float64(30),
	})
	if res.IsError && strings.Contains(res.ForLLM, "every_seconds") {
		t.Errorf("recurring reminder (no command) must not hit the command floor; got: %q", res.ForLLM)
	}
}
