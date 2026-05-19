package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Agentx-network/agentx/pkg/config"
)

// version is set at build time via ldflags:
//
//	-X main.version=0.8.11
// version is the user-visible build version surfaced in the sidebar footer.
// Kept in sync with cmd/agentx-desktop/build/windows/info.json (file_version
// + ProductVersion) and the installer .nsi during release bumps. The default
// here is the fallback when no ldflags override is supplied — wails build
// doesn't auto-inject like the CLI Makefile does. Bumping the version
// requires updating BOTH this constant AND the windows/info.json file.
var version = "0.8.36"

// App struct holds application lifecycle state.
type App struct {
	ctx context.Context
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// AppInfo holds basic application metadata.
type AppInfo struct {
	Version    string `json:"version"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	ConfigPath string `json:"configPath"`
}

// GetAppInfo returns version, OS, and arch information.
func (a *App) GetAppInfo() AppInfo {
	return AppInfo{
		Version:    version,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		ConfigPath: getConfigPath(),
	}
}

// ConfigExists checks whether the AgentX config file exists.
func (a *App) ConfigExists() bool {
	_, err := os.Stat(getConfigPath())
	return err == nil
}

// GetConfigPath returns the path to the AgentX config file.
func (a *App) GetConfigPath() string {
	return getConfigPath()
}

// SetupState describes how far the user has progressed through first-time setup.
type SetupState struct {
	BinaryInstalled bool `json:"binaryInstalled"`
	ConfigExists    bool `json:"configExists"`
	HasAPIKey       bool `json:"hasApiKey"`
	HasChannel      bool `json:"hasChannel"`
}

// GetSetupState checks install + onboard progress in one call.
func (a *App) GetSetupState() SetupState {
	state := SetupState{}

	info := (&InstallerService{}).DetectPlatform()
	state.BinaryInstalled = info.BinaryExists

	cfgPath := getConfigPath()
	if _, err := os.Stat(cfgPath); err == nil {
		state.ConfigExists = true
	}

	if state.ConfigExists {
		cfg, err := loadConfigSafe()
		if err == nil {
			for _, m := range cfg.ModelList {
				if m.APIKey != "" && m.APIKey != "ollama" {
					state.HasAPIKey = true
					break
				}
			}
			if cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token != "" {
				state.HasChannel = true
			} else if cfg.Channels.Discord.Enabled && cfg.Channels.Discord.Token != "" {
				state.HasChannel = true
			} else if cfg.Channels.Slack.Enabled && cfg.Channels.Slack.BotToken != "" {
				state.HasChannel = true
			}
		}
	}

	return state
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agentx", "config.json")
}

func loadConfigSafe() (*config.Config, error) {
	return config.LoadConfig(getConfigPath())
}
