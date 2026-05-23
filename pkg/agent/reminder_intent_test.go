package agent

import (
	"strings"
	"testing"
)

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

		// Phrasings users actually type, beyond "remind me in N":
		{"send a reminder after N", "can you send a reminder after 1 min to drink water", true, 60, "drink water", ""},
		{"set a reminder in N", "set a reminder in 5 minutes to call mom", true, 300, "call mom", ""},
		{"schedule a reminder for N from now", "schedule a reminder for 10 minutes from now", true, 600, "", ""},
		{"create an alert in N", "create an alert in 2 hours", true, 7200, "", ""},
		{"send me a reminder", "send me a reminder in 30 seconds about lunch", true, 30, "lunch", ""},

		// Negative: 'send a reminder email' has the noun but no time → reject.
		{"reminder noun but no time", "send a reminder email later", false, 0, "", ""},
		// Negative: 'after 5 minutes' alone (no reminder intent) → reject.
		{"time without reminder", "after 5 minutes the food was ready", false, 0, "", ""},
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

// The confirmation varies randomly across a small set of phrasings — assert
// structural properties (delay, subject, channel must appear) and that we
// actually see more than one distinct output across many calls (proves the
// randomization is working and the reply doesn't feel robotic).
func TestReminderConfirmationText(t *testing.T) {
	cases := []struct {
		name     string
		subject  string
		delay    int
		ch       string
		mustHave []string
		mustMiss []string
	}{
		{
			name: "with subject, no channel",
			subject: "drink water", delay: 60, ch: "",
			mustHave: []string{"drink water", "1 minute"},
			mustMiss: []string{"(on "},
		},
		{
			name: "no subject, no channel",
			subject: "", delay: 60, ch: "",
			mustHave: []string{"1 minute"},
			mustMiss: []string{"(on "},
		},
		{
			name: "with subject + channel",
			subject: "stretch", delay: 7200, ch: "telegram",
			mustHave: []string{"stretch", "2 hours", "(on Telegram)"},
		},
		{
			name: "seconds",
			subject: "lunch", delay: 30, ch: "",
			mustHave: []string{"lunch", "30 seconds"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			seen := map[string]bool{}
			for i := 0; i < 40; i++ {
				got := reminderConfirmationText(c.subject, c.delay, c.ch)
				for _, must := range c.mustHave {
					if !strings.Contains(got, must) {
						t.Errorf("missing %q in %q", must, got)
					}
				}
				for _, miss := range c.mustMiss {
					if strings.Contains(got, miss) {
						t.Errorf("unexpected %q in %q", miss, got)
					}
				}
				if len(got) > 110 {
					t.Errorf("response too long (%d chars): %q", len(got), got)
				}
				seen[got] = true
			}
			if len(seen) < 2 {
				t.Errorf("expected variation across 40 calls, only saw: %v", seen)
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
