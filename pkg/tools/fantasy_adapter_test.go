package tools

import "testing"

func TestRepairMangledSurrogates(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "water drop emoji corrupted with newlines",
			in:   "Reminder: Time to drink water! \nd83d\ndca7 Stay hydrated!",
			want: "Reminder: Time to drink water! 💧 Stay hydrated!",
		},
		{
			name: "waving hand corrupted (adapter-stage newlines)",
			in:   "Ping! Just checking in...\nd83d\ndc4b",
			want: "Ping! Just checking in...👋",
		},
		{
			name: "plain text with d-words is untouched",
			in:   "deduplicate the dad's dragon data",
			want: "deduplicate the dad's dragon data",
		},
		{
			name: "real emoji is left alone",
			in:   "all good 💧 here",
			want: "all good 💧 here",
		},
		{
			name: "no surrogate-range hex pairs",
			in:   "hex colors #d4d4d4 and #abc123",
			want: "hex colors #d4d4d4 and #abc123",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repairMangledSurrogates(tt.in); got != tt.want {
				t.Errorf("repairMangledSurrogates(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
