package agent

import "testing"

func TestParseReminderIntent(t *testing.T) {
	tests := []struct {
		name        string
		msg         string
		wantOK      bool
		wantSeconds int
		wantSubject string
		wantChannel string
	}{
		{"in minutes with subject", "remind me in 1 min to drink water", true, 60, "drink water", ""},
		{"to-subject before delay", "remind me to call mom in 2 hours", true, 7200, "call mom", ""},
		{"ping telegram", "can you ping me in telegram in 1 min?", true, 60, "", "telegram"},
		{"seconds no subject", "alert me in 30 seconds", true, 30, "", ""},
		{"an hour", "remind me in an hour to stretch", true, 3600, "stretch", ""},
		{"subject with trailing channel", "remind me in 5 minutes to stand up on telegram", true, 300, "stand up", "telegram"},
		{"not a reminder", "what's the weather in 5 minutes from now like?", false, 0, "", ""},
		{"reminder but no time", "remind me to buy milk", false, 0, "", ""},
		{"plain chat", "how are you?", false, 0, "", ""},
		{"zero delay rejected", "remind me in 0 minutes", false, 0, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec, subj, ch, ok := parseReminderIntent(tt.msg)
			if ok != tt.wantOK {
				t.Fatalf("ok=%v want %v (sec=%d subj=%q ch=%q)", ok, tt.wantOK, sec, subj, ch)
			}
			if !ok {
				return
			}
			if sec != tt.wantSeconds {
				t.Errorf("seconds=%d want %d", sec, tt.wantSeconds)
			}
			if subj != tt.wantSubject {
				t.Errorf("subject=%q want %q", subj, tt.wantSubject)
			}
			if ch != tt.wantChannel {
				t.Errorf("channel=%q want %q", ch, tt.wantChannel)
			}
		})
	}
}

func TestHumanizeDelay(t *testing.T) {
	cases := map[int]string{1: "1 second", 30: "30 seconds", 60: "1 minute", 120: "2 minutes", 3600: "1 hour", 7200: "2 hours", 90: "90 seconds"}
	for s, want := range cases {
		if got := humanizeDelay(s); got != want {
			t.Errorf("humanizeDelay(%d)=%q want %q", s, got, want)
		}
	}
}
