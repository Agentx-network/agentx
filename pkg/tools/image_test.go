package tools

import (
	"errors"
	"strings"
	"testing"
)

func TestFriendlyImageError(t *testing.T) {
	cases := []struct {
		name      string
		provider  string
		err       string
		wantUser  string // substring expected in the user-facing message
		notInUser string // substring that must NOT leak into the user message
	}{
		{
			name:      "free tier limit 0 → billing",
			provider:  "gemini",
			err:       "gemini image: You exceeded your current quota ... limit: 0, model: gemini-2.5-flash-preview-image (HTTP 429)",
			wantUser:  "billing",
			notInUser: "http",
		},
		{
			name:     "transient rate limit",
			provider: "openai",
			err:      "openai image: HTTP 429: Too Many Requests",
			wantUser: "rate-limiting",
		},
		{
			name:     "bad key",
			provider: "replicate",
			err:      "replicate image: HTTP 401: invalid_api_key",
			wantUser: "key",
		},
		{
			name:     "generic failure is truncated/one-lined",
			provider: "gemini",
			err:      "gemini image: response contained no image\nmodel may not support image output",
			wantUser: "failed",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			user, llm := friendlyImageError(c.provider, errors.New(c.err))
			if !strings.Contains(strings.ToLower(user), strings.ToLower(c.wantUser)) {
				t.Errorf("user msg %q missing %q", user, c.wantUser)
			}
			if c.notInUser != "" && strings.Contains(strings.ToLower(user), strings.ToLower(c.notInUser)) {
				t.Errorf("user msg %q leaked %q", user, c.notInUser)
			}
			if strings.Contains(user, "\n") {
				t.Errorf("user msg should be single-line, got %q", user)
			}
			if llm == "" {
				t.Error("forLLM should not be empty")
			}
		})
	}
}

func TestProviderLabel(t *testing.T) {
	cases := map[string]string{
		"gemini": "Gemini", "google": "Gemini", "openai": "OpenAI",
		"replicate": "Replicate", "seedance": "Seedance", "": "the provider",
	}
	for in, want := range cases {
		if got := providerLabel(in); got != want {
			t.Errorf("providerLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
