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

// A shell-command cron must require explicit confirmation. Without
// confirmed=true the tool refuses and asks the model to confirm with the user.
// With confirmed=true it goes through (subject to the other guards).
func TestCronAddJob_CommandRequiresConfirmation(t *testing.T) {
	tool := newCronToolForGuardTests(t)

	// One-shot command WITHOUT confirmation → refused.
	res := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "disk check",
		"command":    "df -h",
		"at_seconds": float64(300),
	})
	if !res.IsError {
		t.Fatal("unconfirmed command cron must be refused")
	}
	if !strings.Contains(res.ForUser, "confirm") {
		t.Errorf("user-facing reply should ask for confirmation; got: %q", res.ForUser)
	}
	if !strings.Contains(res.ForLLM, "confirmed=true") {
		t.Errorf("model instruction should mention confirmed=true; got: %q", res.ForLLM)
	}

	// Recurring command WITHOUT confirmation → also refused (and floor doesn't
	// matter — confirmation is required regardless of interval).
	res = tool.Execute(context.Background(), map[string]any{
		"action":        "add",
		"message":       "hourly check",
		"command":       "df -h",
		"every_seconds": float64(3600),
	})
	if !res.IsError || !strings.Contains(res.ForUser, "confirm") {
		t.Errorf("unconfirmed recurring command must be refused; got: %q", res.ForUser)
	}

	// Same call WITH confirmed=true → passes the gate. (Reminder target is
	// desktop via SetContext so the rest of the flow doesn't reject it.)
	tool.SetContext("desktop", "chat")
	res = tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "disk check",
		"command":    "df -h",
		"at_seconds": float64(300),
		"confirmed":  true,
	})
	if res.IsError {
		t.Errorf("confirmed command cron should be accepted; got: %q", res.ForLLM)
	}
}

// Reminders without a command must NOT require the confirmed flag — the gate
// only exists for shell-command crons. Ordinary 'remind me' flows stay simple.
func TestCronAddJob_ReminderNoConfirmationNeeded(t *testing.T) {
	tool := newCronToolForGuardTests(t)
	tool.SetContext("desktop", "chat")
	res := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "drink water",
		"at_seconds": float64(60),
		// no command, no confirmed flag
	})
	if res.IsError {
		t.Errorf("reminder must not require confirmation; got: %q", res.ForLLM)
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
