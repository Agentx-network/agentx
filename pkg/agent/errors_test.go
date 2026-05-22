package agent

import (
	"errors"
	"strings"
	"testing"
)

func TestHumanizeError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantPart string
	}{
		{
			name:     "gemini quota dump",
			err:      errors.New("fantasy agent failed: retry error: too many requests: You exceeded your current quota... limit: 20, model: gemini-2.5-flash. Please retry in 5.7s https://ai.google.dev/gemini-api/docs/rate-limits"),
			wantPart: "free-tier request limit",
		},
		{name: "401 auth", err: errors.New("provider returned 401 unauthorized: invalid_api_key"), wantPart: "rejected the API key"},
		{name: "model missing", err: errors.New(`model "" not found in model_list`), wantPart: "isn't available"},
		{name: "context length", err: errors.New("input exceeds maximum context length tokens"), wantPart: "too long"},
		{name: "network", err: errors.New("dial tcp: connection refused"), wantPart: "Couldn't reach"},
		{name: "unknown", err: errors.New("some weird internal failure xyz"), wantPart: "Something went wrong"},
		{name: "nil", err: nil, wantPart: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := HumanizeError(c.err)
			if c.wantPart == "" {
				if got != "" {
					t.Errorf("want empty, got %q", got)
				}
				return
			}
			if !strings.Contains(got, c.wantPart) {
				t.Errorf("got %q, want substring %q", got, c.wantPart)
			}
			// Must never leak raw URLs or be multi-line.
			if strings.Contains(got, "http") || strings.Contains(got, "\n") {
				t.Errorf("humanized message leaked raw detail: %q", got)
			}
		})
	}
}
