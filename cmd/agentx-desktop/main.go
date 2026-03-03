package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	installer := NewInstallerService()
	dashboard := NewDashboardService()
	configSvc := NewConfigService()
	chatSvc := NewChatService()
	agentSetup := NewAgentSetupService()
	walletSvc := NewWalletService()
	walletSvc.SetDashboard(dashboard)
	registrySvc := NewRegistryService()

	err := wails.Run(&options.App{
		Title:  "AgentX Desktop",
		Width:  1100,
		Height: 720,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			installer.startup(ctx)
			dashboard.startup(ctx)
			configSvc.startup(ctx)
			chatSvc.startup(ctx)
			agentSetup.startup(ctx)
			walletSvc.startup(ctx)
			registrySvc.startup(ctx)
		},
		Bind: []interface{}{
			app,
			installer,
			dashboard,
			configSvc,
			chatSvc,
			agentSetup,
			walletSvc,
			registrySvc,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
