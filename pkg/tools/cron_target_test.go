package tools

import (
	"testing"

	"github.com/Agentx-network/agentx/pkg/config"
)

// Tier 0: a delayed reminder must resolve to a channel that can actually receive
// it. Desktop/CLI can't, so it redirects to a connected push channel — or fails
// honestly rather than scheduling a ping that gets silently dropped.
func TestResolveReminderTarget(t *testing.T) {
	tg := func(enabled bool, allow ...string) *config.Config {
		c := &config.Config{}
		c.Channels.Telegram.Enabled = enabled
		c.Channels.Telegram.AllowFrom = allow
		return c
	}

	t.Run("on telegram already → keep it", func(t *testing.T) {
		tool := &CronTool{cfg: tg(true, "999")}
		ch, id, err := tool.resolveReminderTarget("telegram", "12345")
		if err != nil {
			t.Fatalf("unexpected error: %s", err.ForLLM)
		}
		if ch != "telegram" || id != "12345" {
			t.Errorf("got (%q,%q), want (telegram,12345)", ch, id)
		}
	})

	t.Run("on desktop → redirect to connected telegram + owner id", func(t *testing.T) {
		tool := &CronTool{cfg: tg(true, "1046193410")}
		ch, id, err := tool.resolveReminderTarget("desktop", "chat")
		if err != nil {
			t.Fatalf("unexpected error: %s", err.ForLLM)
		}
		if ch != "telegram" || id != "1046193410" {
			t.Errorf("got (%q,%q), want (telegram,1046193410)", ch, id)
		}
	})

	t.Run("nothing connected → honest error, no schedule", func(t *testing.T) {
		tool := &CronTool{cfg: tg(false)}
		_, _, err := tool.resolveReminderTarget("desktop", "chat")
		if err == nil || !err.IsError {
			t.Fatal("expected an error result when no push channel is connected")
		}
	})
}
