package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const desktopServiceTemplate = `[Unit]
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

const desktopLaunchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
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

func (s *InstallerService) InstallService() error {
	binPath, err := findBinary()
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "linux":
		return installSystemdUnit(binPath)
	case "darwin":
		return installLaunchdPlist(binPath)
	case "windows":
		return installTaskScheduler(binPath)
	default:
		return fmt.Errorf("service installation not supported on %s", runtime.GOOS)
	}
}

// HasWallet returns whether a wallet.json file exists in ~/.agentx.
// The frontend uses this to decide whether to show the export step before uninstall.
func (s *InstallerService) HasWallet() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".agentx", "wallet.json"))
	return err == nil
}

func (s *InstallerService) FullUninstall() error {
	// 1. Remove gateway service (best-effort).
	s.UninstallService()

	// 2. Remove data directory (~/.agentx).
	if home, err := os.UserHomeDir(); err == nil {
		os.RemoveAll(filepath.Join(home, ".agentx"))
	}

	// 3. Remove the CLI binary.
	if binPath, err := findBinary(); err == nil {
		os.Remove(binPath)
	}

	return nil
}

func (s *InstallerService) UninstallService() error {
	switch runtime.GOOS {
	case "linux":
		if err := uninstallSystemdUnit(); err != nil {
			return err
		}
	case "darwin":
		if err := uninstallLaunchdPlist(); err != nil {
			return err
		}
	case "windows":
		if err := uninstallTaskScheduler(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("service uninstallation not supported on %s", runtime.GOOS)
	}
	// Kill any remaining gateway processes
	killGatewayProcesses()
	return nil
}

func killGatewayProcesses() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/IM", "agentx.exe", "/F")
		hideConsoleWindow(cmd)
		cmd.Run()
		return
	}
	exec.Command("pkill", "-f", "agentx gateway").Run()
	exec.Command("pkill", "-f", "agentx.*gateway").Run()
}

func (s *InstallerService) IsServiceRunning() bool {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("systemctl", "--user", "is-active", "agentx-gateway.service")
		err := cmd.Run()
		return err == nil
	case "darwin":
		cmd := exec.Command("launchctl", "list", "com.agentx.gateway")
		err := cmd.Run()
		return err == nil
	case "windows":
		cmd := exec.Command("schtasks", "/Query", "/TN", "AgentXGateway")
		hideConsoleWindow(cmd)
		err := cmd.Run()
		return err == nil
	}
	return false
}

func installSystemdUnit(binPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return err
	}
	unitPath := filepath.Join(unitDir, "agentx-gateway.service")
	content := fmt.Sprintf(desktopServiceTemplate, binPath)
	if err := os.WriteFile(unitPath, []byte(content), 0o644); err != nil {
		return err
	}
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload failed: %s: %w", out, err)
	}
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", "agentx-gateway.service").CombinedOutput(); err != nil {
		return fmt.Errorf("enable failed: %s: %w", out, err)
	}
	return nil
}

func uninstallSystemdUnit() error {
	exec.Command("systemctl", "--user", "disable", "--now", "agentx-gateway.service").Run()
	home, _ := os.UserHomeDir()
	unitPath := filepath.Join(home, ".config", "systemd", "user", "agentx-gateway.service")
	os.Remove(unitPath)
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

func installLaunchdPlist(binPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(plistDir, 0o755); err != nil {
		return err
	}
	plistPath := filepath.Join(plistDir, "com.agentx.gateway.plist")
	content := fmt.Sprintf(desktopLaunchdPlistTemplate, binPath, home, home)
	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return err
	}
	if out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load failed: %s: %w", out, err)
	}
	return nil
}

func uninstallLaunchdPlist() error {
	home, _ := os.UserHomeDir()
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.agentx.gateway.plist")
	exec.Command("launchctl", "unload", plistPath).Run()
	os.Remove(plistPath)
	return nil
}

func installTaskScheduler(binPath string) error {
	cmd := exec.Command(
		"schtasks", "/Create",
		"/SC", "ONLOGON",
		"/TN", "AgentXGateway",
		"/TR", fmt.Sprintf(`"%s" gateway`, binPath),
		"/RL", "LIMITED",
		"/F",
	)
	hideConsoleWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks create failed: %s: %w", out, err)
	}
	return nil
}

func uninstallTaskScheduler() error {
	cmd := exec.Command(
		"schtasks", "/Delete",
		"/TN", "AgentXGateway",
		"/F",
	)
	hideConsoleWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks delete failed: %s: %w", out, err)
	}
	return nil
}
