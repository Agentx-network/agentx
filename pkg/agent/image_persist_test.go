package agent

import (
	"path/filepath"
	"testing"

	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/tools"
)

// persistImageProvider should save an auto-detected provider into the dedicated
// Images config (so it appears on the Config page), and must not overwrite an
// entry the user already configured.
func TestPersistImageProvider(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	persistImageProvider(cfgPath, tools.ImageProvider{Provider: "gemini", Model: "gemini-2.5-flash-image", APIKey: "AIza-chat-key"})

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got, ok := cfg.Tools.Image.Providers["gemini"]
	if !ok || got.APIKey != "AIza-chat-key" {
		t.Fatalf("gemini not persisted correctly: %+v (ok=%v)", got, ok)
	}

	// Re-persist with a different key → must NOT overwrite the existing entry.
	persistImageProvider(cfgPath, tools.ImageProvider{Provider: "gemini", APIKey: "different-key"})
	cfg2, _ := config.LoadConfig(cfgPath)
	if cfg2.Tools.Image.Providers["gemini"].APIKey != "AIza-chat-key" {
		t.Error("persist overwrote an existing provider entry; it should be idempotent")
	}

	// Empty provider/key is a no-op (no panic, nothing written).
	persistImageProvider(cfgPath, tools.ImageProvider{})
}
