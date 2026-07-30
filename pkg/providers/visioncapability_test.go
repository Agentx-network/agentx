package providers

import "testing"

func TestProviderSupportsVision(t *testing.T) {
	yes := []struct{ p, m string }{
		{"anthropic", "claude-3-5-sonnet"},
		{"anthropic", "claude-sonnet-4"},
		{"gemini", "gemini-2.5-flash"},
		{"google", "gemini-1.5-pro"},
		{"openai", "gpt-4o"},
		{"openai", "gpt-4.1"},
		{"openai", "gpt-5.2"},
		{"openai", "o3"},
		{"openrouter", "qwen2-vl-7b"},
		{"openrouter", "auto"}, // router selects a vision model per-request
	}
	for _, c := range yes {
		if !ProviderSupportsVision(c.p, c.m) {
			t.Errorf("expected %s/%s to support vision", c.p, c.m)
		}
	}

	no := []struct{ p, m string }{
		{"anthropic", "claude-2.1"},
		{"anthropic", "claude-instant-1"},
		{"gemini", "gemini-pro"},
		{"openai", "gpt-3.5-turbo"},
		{"groq", "llama-3.3-70b"},
		{"deepseek", "deepseek-chat"},
		{"cerebras", "llama3.1-8b"},
	}
	for _, c := range no {
		if ProviderSupportsVision(c.p, c.m) {
			t.Errorf("expected %s/%s to NOT support vision", c.p, c.m)
		}
	}
}

func TestProviderSupportsAudioInput(t *testing.T) {
	if !ProviderSupportsAudioInput("gemini", "gemini-2.5-flash") {
		t.Error("expected gemini to support audio input")
	}
	if !ProviderSupportsAudioInput("openai", "gpt-4o-audio-preview") {
		t.Error("expected gpt-4o-audio to support audio input")
	}
	if ProviderSupportsAudioInput("anthropic", "claude-sonnet-4") {
		t.Error("anthropic does not support audio input")
	}
	if ProviderSupportsAudioInput("openai", "gpt-4o") {
		t.Error("plain gpt-4o does not take input_audio")
	}
}
