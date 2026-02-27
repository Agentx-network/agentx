package onboard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const serviceTemplate = `[Unit]
Description=AgentX Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s gateway
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
`

const launchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.agentx.gateway</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>gateway</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s/.agentx/gateway.log</string>
	<key>StandardErrorPath</key>
	<string>%s/.agentx/gateway.err.log</string>
</dict>
</plist>
`

func isSystemdAvailable() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func isLaunchdAvailable() bool {
	return runtime.GOOS == "darwin"
}

func isServiceAvailable() bool {
	return isSystemdAvailable() || isLaunchdAvailable()
}

func findAgentxBinary() (string, error) {
	binPath, err := exec.LookPath("agentx")
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot find agentx binary: %w", err)
		}
		binPath = filepath.Join(home, ".local", "bin", "agentx")
		if _, err := os.Stat(binPath); err != nil {
			return "", fmt.Errorf("cannot find agentx binary in PATH or %s", binPath)
		}
	}
	return binPath, nil
}

func installGatewayService() error {
	if isLaunchdAvailable() {
		return installLaunchdService()
	}
	return installSystemdService()
}

func installSystemdService() error {
	binPath, err := findAgentxBinary()
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return fmt.Errorf("cannot create systemd user directory: %w", err)
	}

	unitPath := filepath.Join(unitDir, "agentx-gateway.service")
	content := fmt.Sprintf(serviceTemplate, binPath)
	if err := os.WriteFile(unitPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cannot write unit file: %w", err)
	}

	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload failed: %s: %w", out, err)
	}
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", "agentx-gateway.service").CombinedOutput(); err != nil {
		return fmt.Errorf("enable failed: %s: %w", out, err)
	}

	return nil
}

func installLaunchdService() error {
	binPath, err := findAgentxBinary()
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(plistDir, 0o755); err != nil {
		return fmt.Errorf("cannot create LaunchAgents directory: %w", err)
	}

	plistPath := filepath.Join(plistDir, "com.agentx.gateway.plist")
	content := fmt.Sprintf(launchdPlistTemplate, binPath, home, home)
	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cannot write plist file: %w", err)
	}

	if out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load failed: %s: %w", out, err)
	}

	return nil
}
