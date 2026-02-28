package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Agentx-network/agentx/pkg/config"
)

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
		Version:    "0.1.0",
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
}

// GetSetupState checks install + onboard progress in one call.
func (a *App) GetSetupState() SetupState {
	state := SetupState{}

	// Check binary
	info := (&InstallerService{}).DetectPlatform()
	state.BinaryInstalled = info.BinaryExists

	// Check config
	cfgPath := getConfigPath()
	if _, err := os.Stat(cfgPath); err == nil {
		state.ConfigExists = true
	}

	// Check if any model has an API key configured
	if state.ConfigExists {
		cfg, err := loadConfigSafe()
		if err == nil {
			for _, m := range cfg.ModelList {
				if m.APIKey != "" && m.APIKey != "ollama" {
					state.HasAPIKey = true
					break
				}
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
