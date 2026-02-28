# AgentX Desktop

A Wails v2 desktop application (Go backend + React/TypeScript frontend) that provides a graphical installer, status dashboard, and configuration panel for AgentX.

## Architecture

```
cmd/agentx-desktop/
├── main.go                  # Wails entry point, binds all services
├── app.go                   # App lifecycle (GetAppInfo, ConfigExists)
├── installer.go             # Download/install binary from GitHub releases
├── installer_linux.go       # Linux: install dir (~/.local/bin), PATH setup
├── installer_darwin.go      # macOS: install dir, PATH setup
├── installer_windows.go     # Windows: install dir, registry PATH
├── dashboard.go             # Health polling, logs, gateway start/stop/restart
├── configui.go              # Config load/save bridge to pkg/config
├── service.go               # Service install/uninstall (systemd/launchd)
├── wails.json               # Wails project configuration
├── build/
│   └── bin/                 # Built binary output
└── frontend/
    ├── index.html           # HTML entry
    ├── package.json         # Node dependencies
    ├── tsconfig.json        # TypeScript config
    ├── vite.config.ts       # Vite bundler config
    ├── tailwind.config.js   # Tailwind theme (neon pink/purple/cyan)
    ├── postcss.config.js    # PostCSS + Tailwind
    └── src/
        ├── main.tsx         # React entry
        ├── App.tsx          # Layout shell + page router
        ├── index.css        # Tailwind base + glass-morphism styles
        ├── lib/types.ts     # TS interfaces mirroring Go structs
        ├── pages/
        │   ├── InstallerPage.tsx   # Download & install AgentX binary
        │   ├── DashboardPage.tsx   # Health, channels, models, logs
        │   └── ConfigPage.tsx      # Full config editor + quick setup
        └── components/
            ├── Sidebar.tsx          # Navigation sidebar
            ├── StatusBadge.tsx      # Running/stopped indicator
            ├── DownloadProgress.tsx  # Install progress bar
            ├── HealthCard.tsx       # Gateway health card
            ├── ChannelsCard.tsx     # Active channels list
            ├── ModelsCard.tsx       # Model list
            ├── LogViewer.tsx        # Log tail viewer
            ├── ProvidersSection.tsx  # Model list CRUD editor
            ├── ChannelsSection.tsx   # Channel toggles
            ├── AgentSection.tsx     # Agent defaults editor
            └── ui/
                ├── NeonButton.tsx   # Themed button (primary/ghost/danger)
                ├── NeonInput.tsx    # Themed text input
                ├── NeonSelect.tsx   # Themed select dropdown
                ├── NeonToggle.tsx   # Toggle switch
                ├── NeonCard.tsx     # Glass card (collapsible)
                └── Toast.tsx        # Notification toast
```

## Prerequisites

### System Dependencies (Linux - Ubuntu/Debian)

```bash
sudo apt install -y libgtk-3-dev libwebkit2gtk-4.1-dev build-essential pkg-config
```

For other distros see: https://wails.io/docs/gettingstarted/installation#platform-specific-dependencies

### Toolchain

- **Go** >= 1.21
- **Node.js** >= 18 with npm
- **Wails CLI** v2:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Verify everything is ready:

```bash
wails doctor
```

All dependencies should show **Installed**. Fix any that show **Not Found** before proceeding.

## Development

### First Time Setup

```bash
# From repo root
cd cmd/agentx-desktop

# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in dev mode (hot reload for frontend, auto-rebuild for backend)
wails dev -tags webkit2_41
```

This opens the desktop window. Frontend changes hot-reload instantly; Go changes trigger a rebuild.

### Build for Production

```bash
# From cmd/agentx-desktop/
wails build -tags webkit2_41
```

Binary output: `build/bin/agentx-desktop`

### Using Makefile (from repo root)

```bash
make desktop-dev     # wails dev (hot reload)
make desktop-build   # wails build (production)
```

## Go Backend Services

Four services are bound to the Wails frontend:

### App (`app.go`)
- `GetAppInfo()` — returns version, OS, arch, config path
- `ConfigExists()` — checks if `~/.agentx/config.json` exists
- `GetConfigPath()` — returns config file path

### InstallerService (`installer.go` + platform files + `service.go`)
- `DetectPlatform()` — returns OS, arch, install dir, whether binary exists, installed version
- `GetLatestRelease()` — fetches latest tag from GitHub releases API
- `InstallBinary()` — downloads `agentx-{os}-{arch}` from GitHub releases, emits `download:progress` events with percent
- `InstallService()` — installs systemd unit (Linux) or launchd plist (macOS)
- `UninstallService()` — removes the service and kills any running gateway processes
- `IsServiceRunning()` — checks if the service is active

### DashboardService (`dashboard.go`)
- `GetStatus()` — polls `http://{host}:{port}/health`, loads config for channel/model info
- `GetLogs(lines)` — reads `~/.agentx/gateway.log` or `journalctl`
- `StartGateway()` — starts via systemd/launchd, falls back to direct process spawn
- `StopGateway()` — stops via service manager + kills any direct processes
- `RestartGateway()` — stop then start

### ConfigService (`configui.go`)
- `GetConfig()` / `SaveConfig(cfg)` — reads/writes `~/.agentx/config.json` via `pkg/config`
- `GetModelList()` / `AddModel()` / `UpdateModel(index, model)` / `RemoveModel(index)` — model CRUD
- `SetChannelEnabled(channel, enabled)` — toggle a channel on/off
- `GetAgentDefaults()` / `UpdateAgentDefaults(defaults)` — agent settings
- `GetAvailableProviders()` — returns provider catalog (OpenAI, Anthropic, Gemini, etc.)
- `QuickSetupProvider(providerID, apiKey)` — one-click provider configuration

**Design**: No in-memory cache. Every Get reads from disk, every Save writes to disk. CLI and desktop always see the same config state.

## Frontend

### Theme
- Dark background: `#121218`
- Neon Pink: `#FF0092`, Neon Purple: `#AE00FF`, Neon Cyan: `#00FFFF`
- Glass-morphism cards with backdrop blur and subtle neon borders

### Pages

**Installer** — Detects OS/arch, shows install button with progress bar, offers "Install as Service" after binary install.

**Dashboard** — Polls gateway health every 5s. Shows health status (green/red), uptime, enabled channels, configured models, log viewer. Start/Stop/Restart buttons.

**Config** — Collapsible sections:
- Quick Setup (provider picker + API key)
- Models (editable table)
- Channels (toggle switches)
- Agent Defaults (workspace, model, temperature, max tokens)

### State
React hooks (`useState`/`useEffect`) per page — no external state library needed for 3 pages.

## Build Integration

### GoReleaser

The `.goreleaser.yaml` includes an `agentx-desktop` build target:
- `CGO_ENABLED=1` (required for webview)
- Platforms: `linux/amd64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- Frontend is built via pre-hooks (`npm install` + `npm run build`)

### Notes

- The desktop app is a **separate binary** from the CLI (`cmd/agentx/`). They share `pkg/config` and `pkg/health`.
- The `webkit2_41` build tag is needed on Ubuntu 24.04+ (uses WebKit2GTK 4.1 instead of 4.0).
- `go build ./...` and `go test ./...` from the repo root still pass — existing tests are unaffected.
- The `frontend/dist/` directory must exist for the Go embed directive. It is created by `npm run build` or `wails build`.
