package providers

import "testing"

func TestIsLowCapabilityModel(t *testing.T) {
	weak := []string{
		"cerebras/llama-3.1-8b",
		"groq/llama-3.1-8b-instant",
		"groq/gemma2-9b-it",
		"openai/gpt-4o-mini",
		"gpt-4.1-nano",
		"some-tiny-model",
		"qwen-3b",
	}
	for _, m := range weak {
		if !IsLowCapabilityModel(m) {
			t.Errorf("expected %q to be flagged low-capability", m)
		}
	}
	strong := []string{
		"cerebras/qwen-3-235b-a22b-instruct-2507",
		"cerebras/llama-3.3-70b",
		"groq/llama-3.3-70b-versatile",
		"gemini/gemini-2.5-flash",
		"anthropic/claude-sonnet-4-6",
		"openai/gpt-4o",
		"deepseek/deepseek-chat",
		"",
	}
	for _, m := range strong {
		if IsLowCapabilityModel(m) {
			t.Errorf("did NOT expect %q to be flagged low-capability", m)
		}
	}
}
