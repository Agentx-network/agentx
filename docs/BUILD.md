# Building AgentX — CLI & Desktop

This document covers how to build **both** the CLI binaries and Desktop app installers, and how to upload them to a GitHub release.

---

## Prerequisites

| Tool | Required for | Install |
|------|-------------|---------|
| **Go** >= 1.21 | CLI + Desktop | [go.dev](https://go.dev/dl/) |
| **Node.js** >= 18 + npm | Desktop frontend | [nodejs.org](https://nodejs.org/) |
| **Wails CLI** v2 | Desktop builds | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| **gh** (GitHub CLI) | Uploading to releases | [cli.github.com](https://cli.github.com/) |

### Platform-specific deps (Desktop only)

| Platform | Packages |
|----------|----------|
| **Ubuntu/Debian** | `sudo apt install -y libwebkit2gtk-4.1-dev libgtk-3-dev build-essential pkg-config` |
| **Fedora** | `sudo dnf install webkit2gtk4.1-devel gtk3-devel` |
| **macOS** | None (WebView built-in) |
| **Windows** | NSIS for installer: `choco install nsis -y` |

---

## 1. CLI Binaries

The CLI uses `CGO_ENABLED=0` — can be cross-compiled from **any** platform.

### Build all platforms at once (Makefile)

```bash
make build-all
```

Output in `build/`:
- `agentx-linux-amd64`, `agentx-linux-arm64`, `agentx-linux-armv7`
- `agentx-linux-riscv64`, `agentx-linux-loong64`
- `agentx-darwin-amd64`, `agentx-darwin-arm64`
- `agentx-windows-amd64.exe`

### Build all platforms manually

```bash
VERSION="0.5.0"
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
GOVERSION=$(go version | awk '{print $3}')
INTERNAL="github.com/Agentx-network/agentx/cmd/agentx/internal"
LDFLAGS="-s -w \
  -X ${INTERNAL}.version=${VERSION} \
  -X ${INTERNAL}.gitCommit=${COMMIT} \
  -X ${INTERNAL}.buildTime=${DATE} \
  -X ${INTERNAL}.goVersion=${GOVERSION}"

CGO_ENABLED=0 GOOS=linux   GOARCH=amd64         go build -tags stdjson -ldflags="$LDFLAGS" -o agentx-linux-amd64       ./cmd/agentx
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64          go build -tags stdjson -ldflags="$LDFLAGS" -o agentx-linux-arm64       ./cmd/agentx
CGO_ENABLED=0 GOOS=linux   GOARCH=arm    GOARM=7 go build -tags stdjson -ldflags="$LDFLAGS" -o agentx-linux-armv7       ./cmd/agentx
CGO_ENABLED=0 GOOS=linux   GOARCH=riscv64        go build -tags stdjson -ldflags="$LDFLAGS" -o agentx-linux-riscv64     ./cmd/agentx
CGO_ENABLED=0 GOOS=linux   GOARCH=loong64        go build -tags stdjson -ldflags="$LDFLAGS" -o agentx-linux-loong64     ./cmd/agentx
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64          go build -tags stdjson -ldflags="$LDFLAGS" -o agentx-darwin-arm64      ./cmd/agentx
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64          go build -tags stdjson -ldflags="$LDFLAGS" -o agentx-darwin-amd64      ./cmd/agentx
CGO_ENABLED=0 GOOS=windows GOARCH=amd64          go build -tags stdjson -ldflags="$LDFLAGS" -o agentx-windows-amd64.exe ./cmd/agentx
```

### Upload CLI binaries to release

```bash
TAG="v0.5.0"
for f in agentx-linux-* agentx-darwin-* agentx-windows-*; do
  gh release upload "$TAG" "$f" --clobber
done
```

---

## 2. Desktop App

The desktop app uses **Wails v2** (Go + React). It requires `CGO_ENABLED=1` and platform-native libraries, so it **cannot be cross-compiled** — each platform must build on its own OS.

### Build for current platform

```bash
cd cmd/agentx-desktop
wails build -clean -ldflags="-s -w" -tags webkit2_41   # Linux needs webkit2_41 tag
```

Or from repo root:

```bash
make desktop-build
```

Binary output: `cmd/agentx-desktop/build/bin/agentx-desktop`

### Linux — create .deb package

```bash
# 1. Build
cd cmd/agentx-desktop
wails build -clean -ldflags="-s -w" -tags webkit2_41
cd ../..

# 2. Package
VERSION="0.5.0"
mkdir -p deb_pkg/DEBIAN
mkdir -p deb_pkg/usr/bin
mkdir -p deb_pkg/usr/share/applications
mkdir -p deb_pkg/usr/share/icons/hicolor/256x256/apps

cp cmd/agentx-desktop/build/bin/agentx-desktop deb_pkg/usr/bin/
cp assets/agentx-desktop-icon.png deb_pkg/usr/share/icons/hicolor/256x256/apps/agentx-desktop.png

cat > deb_pkg/usr/share/applications/agentx-desktop.desktop <<'EOF'
[Desktop Entry]
Name=AgentX Desktop
Comment=AI Agent Dashboard
Exec=/usr/bin/agentx-desktop
Icon=agentx-desktop
Terminal=false
Type=Application
Categories=Utility;Development;
EOF

cat > deb_pkg/DEBIAN/control <<EOF
Package: agentx-desktop
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Depends: libwebkit2gtk-4.1-0, libgtk-3-0
Maintainer: AgentX Network <dev@agentx.network>
Homepage: https://agentx.network
Description: AgentX Desktop App
 GUI dashboard to install, configure, and manage AgentX agents.
EOF

dpkg-deb --build deb_pkg AgentX-Desktop-amd64.deb

# 3. Upload
gh release upload v${VERSION} AgentX-Desktop-amd64.deb --clobber
```

### macOS — create .dmg

```bash
# 1. Build (run on macOS)
cd cmd/agentx-desktop
wails build -clean -ldflags="-s -w"
cd ../..

# 2. Package
ARCH=$(uname -m)  # arm64 or x86_64
if [ -d cmd/agentx-desktop/build/bin/agentx-desktop.app ]; then
  # Wails produced .app bundle
  hdiutil create -volname "AgentX Desktop" \
    -srcfolder cmd/agentx-desktop/build/bin/agentx-desktop.app \
    -ov -format UDZO "AgentX-Desktop-${ARCH}.dmg"
else
  # Fallback: wrap binary
  mkdir -p dmg_staging
  cp cmd/agentx-desktop/build/bin/agentx-desktop dmg_staging/
  ln -s /Applications dmg_staging/Applications
  hdiutil create -volname "AgentX Desktop" \
    -srcfolder dmg_staging -ov -format UDZO "AgentX-Desktop-${ARCH}.dmg"
fi

# 3. Upload
gh release upload v0.5.0 "AgentX-Desktop-${ARCH}.dmg" --clobber
```

### Windows — create Setup.exe (NSIS)

```bash
# 1. Install NSIS (if not installed)
choco install nsis -y

# 2. Build with NSIS flag (run on Windows)
cd cmd/agentx-desktop
wails build -clean -nsis -ldflags="-s -w"
cd ../..

# 3. The installer is at:
#    cmd/agentx-desktop/build/bin/agentx-desktop-amd64-installer.exe
# Rename it:
cp cmd/agentx-desktop/build/bin/agentx-desktop-amd64-installer.exe AgentX-Desktop-Setup.exe

# 4. Upload
gh release upload v0.5.0 AgentX-Desktop-Setup.exe --clobber
```

---

## 3. Release asset naming

### CLI binaries (raw, no extension on Linux/macOS)
| File | Platform |
|------|----------|
| `agentx-linux-amd64` | Linux x86_64 |
| `agentx-linux-arm64` | Linux ARM64 |
| `agentx-linux-armv7` | Linux ARMv7 |
| `agentx-linux-riscv64` | Linux RISC-V |
| `agentx-linux-loong64` | Linux LoongArch |
| `agentx-darwin-amd64` | macOS Intel |
| `agentx-darwin-arm64` | macOS Apple Silicon |
| `agentx-windows-amd64.exe` | Windows x64 |

### CLI installers (built by `build-installers.yml`)
| File | Platform |
|------|----------|
| `AgentX-Setup-v{VERSION}.exe` | Windows (Inno Setup) |
| `AgentX-{VERSION}-amd64.pkg` | macOS Intel (.pkg) |
| `AgentX-{VERSION}-arm64.pkg` | macOS ARM (.pkg) |

### Desktop app installers (built by `build-installers.yml`)
| File | Platform |
|------|----------|
| `AgentX-Desktop-Setup.exe` | Windows (NSIS) |
| `AgentX-Desktop-x86_64.dmg` | macOS Intel |
| `AgentX-Desktop-arm64.dmg` | macOS Apple Silicon |
| `AgentX-Desktop-amd64.deb` | Ubuntu/Debian |

---

## 4. CI/CD workflows

| Workflow | Trigger | What it builds |
|----------|---------|---------------|
| `release.yml` | `workflow_dispatch` (manual tag) | CLI binaries + Docker via GoReleaser |
| `build-installers.yml` | `release:published` or `workflow_dispatch` | CLI installers (.exe, .pkg) + Desktop installers (.exe, .dmg, .deb) |
| `pages.yml` | Push to `docs/**` | Install page at `install.agentx.network` |

### Manually trigger desktop rebuild

```bash
gh workflow run build-installers.yml -f tag=v0.5.0
```

### Full release flow

1. `gh workflow run release.yml -f tag=v0.6.0` — creates tag, builds CLI binaries + Docker
2. `build-installers.yml` triggers automatically on release publish — builds all installers
3. Or re-trigger manually: `gh workflow run build-installers.yml -f tag=v0.6.0`

---

## 5. Key build differences

| | CLI (`cmd/agentx`) | Desktop (`cmd/agentx-desktop`) |
|---|---|---|
| **CGO** | `CGO_ENABLED=0` | `CGO_ENABLED=1` |
| **Cross-compile** | Yes (any OS builds all targets) | No (must build on native OS) |
| **Frontend** | None | React + Vite (embedded via `//go:embed`) |
| **Build tool** | `go build` | `wails build` |
| **Linux tag** | `stdjson` | `webkit2_41` |
| **Dependencies** | None | WebKit2GTK (Linux), WebView2 (Windows) |
