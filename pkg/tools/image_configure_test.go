package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Agentx-network/agentx/pkg/config"
)

func TestConfigureImageProviderPersists(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	tool := NewConfigureImageProviderTool(cfgPath)

	res := tool.Execute(context.Background(), map[string]any{
		"provider": "replicate",
		"api_key":  "r8_test_token",
	})
	if res.IsError {
		t.Fatalf("expected success, got error: %s", res.ForLLM)
	}

	// The key must be written to the same config the gateway/desktop read.
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	got, ok := cfg.Tools.Image.Providers["replicate"]
	if !ok {
		t.Fatal("replicate not saved to tools.image.providers")
	}
	if got.APIKey != "r8_test_token" {
		t.Errorf("saved key = %q, want r8_test_token", got.APIKey)
	}
}

func TestConfigureImageProviderRejectsNonImageProvider(t *testing.T) {
	tool := NewConfigureImageProviderTool(filepath.Join(t.TempDir(), "config.json"))
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "anthropic",
		"api_key":  "sk-test",
	})
	if !res.IsError {
		t.Error("expected error for non-image provider anthropic")
	}
}

func TestConfigureImageProviderRequiresKey(t *testing.T) {
	tool := NewConfigureImageProviderTool(filepath.Join(t.TempDir(), "config.json"))
	res := tool.Execute(context.Background(), map[string]any{"provider": "gemini"})
	if !res.IsError {
		t.Error("expected error when api_key missing")
	}
}

// TestImageGenerateResolvesLive verifies the resolver is consulted on every
// call, so a provider added after the tool is built is picked up without a
// rebuild. Uses Replicate with an empty model, which errors synchronously
// (no network) — enough to prove we got past the "no provider" branch.
func TestImageGenerateResolvesLive(t *testing.T) {
	var live []ImageProvider
	calls := 0
	tool := NewImageGenerateTool(func() []ImageProvider {
		calls++
		return live
	}, t.TempDir())

	// No providers yet → blocked with setup guidance.
	res := tool.Execute(context.Background(), map[string]any{"prompt": "a cat"})
	if !res.IsError || !strings.Contains(res.ForLLM, "No image-capable provider is configured") {
		t.Fatalf("expected no-provider guidance, got: %s", res.ForLLM)
	}

	// Add a provider live; the same tool instance must now use it.
	live = append(live, ImageProvider{Provider: "replicate", Model: "", APIKey: "k"})
	res = tool.Execute(context.Background(), map[string]any{"prompt": "a cat"})
	if strings.Contains(res.ForLLM, "No image-capable provider is configured") {
		t.Errorf("resolver not consulted; still reports no provider: %s", res.ForLLM)
	}
	if calls != 2 {
		t.Errorf("resolver called %d times, want 2 (once per Execute)", calls)
	}
}
