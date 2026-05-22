package tools

import (
	"testing"

	"github.com/Agentx-network/agentx/pkg/config"
)

// A delayed reminder must resolve to a channel that can actually receive it.
// The desktop is now a valid local target (the gateway queues it and the app
// polls), so a plain "remind me…" from the desktop fires back into that chat;
// an explicitly-named channel (e.g. "ping me on Telegram") routes there instead.
func TestResolveReminderTarget(t *testing.T) {
	tg := func(enabled bool, allow ...string) *config.Config {
		c := &config.Config{}
		c.Channels.Telegram.Enabled = enabled
		c.Channels.Telegram.AllowFrom = allow
		return c
	}

	t.Run("on telegram already → keep it", func(t *testing.T) {
		tool := &CronTool{cfg: tg(true, "999")}
		ch, id, err := tool.resolveReminderTarget("telegram", "12345", "")
		if err != nil {
			t.Fatalf("unexpected error: %s", err.ForLLM)
		}
		if ch != "telegram" || id != "12345" {
			t.Errorf("got (%q,%q), want (telegram,12345)", ch, id)
		}
	})

	t.Run("on desktop, no channel named → deliver in the desktop chat", func(t *testing.T) {
		tool := &CronTool{cfg: tg(true, "1046193410")}
		ch, id, err := tool.resolveReminderTarget("desktop", "chat", "")
		if err != nil {
			t.Fatalf("unexpected error: %s", err.ForLLM)
		}
		if ch != "desktop" || id != "chat" {
			t.Errorf("got (%q,%q), want (desktop,chat)", ch, id)
		}
	})

	t.Run("on desktop, user names telegram → route to telegram owner", func(t *testing.T) {
		tool := &CronTool{cfg: tg(true, "1046193410")}
		ch, id, err := tool.resolveReminderTarget("desktop", "chat", "telegram")
		if err != nil {
			t.Fatalf("unexpected error: %s", err.ForLLM)
		}
		if ch != "telegram" || id != "1046193410" {
			t.Errorf("got (%q,%q), want (telegram,1046193410)", ch, id)
		}
	})

	t.Run("user names telegram but it's not connected → honest error", func(t *testing.T) {
		tool := &CronTool{cfg: tg(false)}
		_, _, err := tool.resolveReminderTarget("desktop", "chat", "telegram")
		if err == nil || !err.IsError {
			t.Fatal("expected an error when the requested channel isn't connected")
		}
	})

	t.Run("desktop is always deliverable even with no push channel", func(t *testing.T) {
		tool := &CronTool{cfg: tg(false)}
		ch, _, err := tool.resolveReminderTarget("desktop", "chat", "")
		if err != nil {
			t.Fatalf("desktop should be deliverable without a push channel, got error: %s", err.ForLLM)
		}
		if ch != "desktop" {
			t.Errorf("got %q, want desktop", ch)
		}
	})
}
