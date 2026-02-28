package uninstall

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

func NewUninstallCommand() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"remove"},
		Short:   "Remove agentx and all its data from this system",
		Long: `Removes the agentx binary, configuration, workspace, auth credentials,
and the systemd gateway service. Each step is best-effort — failures are
reported but do not stop the remaining cleanup.`,
		Example: `  agentx uninstall        # interactive confirmation
  agentx uninstall --yes  # skip confirmation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				var confirm bool
				err := huh.NewConfirm().
					Title("Uninstall AgentX?").
					Description("This will remove the binary, all config/data (~/.agentx), and the gateway service.").
					Affirmative("Yes, uninstall").
					Negative("Cancel").
					Value(&confirm).
					Run()
				if err != nil {
					return fmt.Errorf("prompt failed: %w", err)
				}
				if !confirm {
					fmt.Println("Uninstall cancelled.")
					return nil
				}
			}
			return runUninstall()
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runUninstall() error {
	var warnings []string

	if err := removeGatewayService(); err != nil {
		warnings = append(warnings, fmt.Sprintf("gateway service: %v", err))
	} else {
		fmt.Println("  Removed gateway service")
	}

	if err := removeDesktopApp(); err != nil {
		warnings = append(warnings, fmt.Sprintf("desktop app: %v", err))
	} else {
		fmt.Println("  Removed agentx-desktop")
	}

	if err := removeDataDir(); err != nil {
		warnings = append(warnings, fmt.Sprintf("data directory: %v", err))
	} else {
		fmt.Println("  Removed ~/.agentx")
	}

	if err := removeBinary(); err != nil {
		warnings = append(warnings, fmt.Sprintf("binary: %v", err))
	} else {
		fmt.Println("  Removed agentx binary")
	}

	if len(warnings) > 0 {
		fmt.Println("\nCompleted with warnings:")
		for _, w := range warnings {
			fmt.Printf("  - %s\n", w)
		}
	} else {
		fmt.Println("\nAgentX has been fully uninstalled.")
	}

	return nil
}

func removeGatewayService() error {
	if runtime.GOOS == "darwin" {
		return removeLaunchdService()
	}
	return removeSystemdService()
}

func removeSystemdService() error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemctl not found, skipping service removal")
	}

	// Best-effort stop & disable — ignore errors (service may not be running).
	_ = exec.Command("systemctl", "--user", "disable", "--now", "agentx-gateway.service").Run()

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	unitPath := filepath.Join(home, ".config", "systemd", "user", "agentx-gateway.service")
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file: %w", err)
	}

	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()

	return nil
}

func removeLaunchdService() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.agentx.gateway.plist")

	// Best-effort unload — ignore errors (service may not be loaded).
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist file: %w", err)
	}

	return nil
}

func removeDataDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	dir := filepath.Join(home, ".agentx")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist", dir)
	}

	return os.RemoveAll(dir)
}

func removeDesktopApp() error {
	// Best-effort kill any running agentx-desktop processes.
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/IM", "agentx-desktop.exe", "/F").Run()
	} else {
		_ = exec.Command("pkill", "-f", "agentx-desktop").Run()
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	// Check known install locations for the desktop binary.
	binaryName := "agentx-desktop"
	if runtime.GOOS == "windows" {
		binaryName = "agentx-desktop.exe"
	}

	candidates := []string{
		filepath.Join(home, ".local", "bin", binaryName),
		filepath.Join("/usr", "local", "bin", binaryName),
	}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, filepath.Join("/Applications", "AgentX Desktop.app"))
	}

	removed := false
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("remove %s: %w", path, err)
			}
			removed = true
		}
	}

	if !removed {
		return fmt.Errorf("agentx-desktop binary not found in known locations")
	}
	return nil
}

func removeBinary() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	// Resolve symlinks so we remove the real file.
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		real = exe
	}

	return os.Remove(real)
}
