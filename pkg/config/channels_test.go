package config

import "testing"

func tgConfig(enabled bool, allow ...string) *Config {
	c := &Config{}
	c.Channels.Telegram.Enabled = enabled
	c.Channels.Telegram.AllowFrom = allow
	return c
}

func TestChannelEnabled(t *testing.T) {
	c := tgConfig(true, "123")
	if !c.ChannelEnabled("telegram") || !c.ChannelEnabled("Telegram") {
		t.Error("telegram should be enabled (case-insensitive)")
	}
	if c.ChannelEnabled("discord") || c.ChannelEnabled("nonsense") {
		t.Error("disabled/unknown channels should report false")
	}
}

func TestOwnerChatID(t *testing.T) {
	cases := []struct {
		allow []string
		want  string
	}{
		{[]string{"1046193410"}, "1046193410"},
		{[]string{"1046193410|alice"}, "1046193410"}, // id|username → id
		{[]string{"*", "777"}, "777"},                // skip wildcard
		{[]string{}, ""},
		{[]string{"*"}, ""},
	}
	for _, tc := range cases {
		got := tgConfig(true, tc.allow...).OwnerChatID("telegram")
		if got != tc.want {
			t.Errorf("OwnerChatID(%v) = %q, want %q", tc.allow, got, tc.want)
		}
	}
}

func TestFirstConnectedPushChannel(t *testing.T) {
	// Enabled telegram with an owner ID → that's the target.
	ch, owner := tgConfig(true, "1046193410").FirstConnectedPushChannel()
	if ch != "telegram" || owner != "1046193410" {
		t.Errorf("got (%q,%q), want (telegram,1046193410)", ch, owner)
	}

	// Enabled but no owner ID → not usable as a target.
	if ch, _ := tgConfig(true).FirstConnectedPushChannel(); ch != "" {
		t.Errorf("expected no target when allow_from empty, got %q", ch)
	}

	// Disabled → no target.
	if ch, _ := tgConfig(false, "123").FirstConnectedPushChannel(); ch != "" {
		t.Errorf("expected no target when channel disabled, got %q", ch)
	}
}

func TestIsPushChannel(t *testing.T) {
	c := &Config{}
	for _, ch := range []string{"telegram", "discord", "slack"} {
		if !c.IsPushChannel(ch) {
			t.Errorf("%s should be a push channel", ch)
		}
	}
	for _, ch := range []string{"desktop", "cli", ""} {
		if c.IsPushChannel(ch) {
			t.Errorf("%s should NOT be a push channel", ch)
		}
	}
}
