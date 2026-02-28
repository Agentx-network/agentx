//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func getInstallDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}

func updatePATH(dir string) error {
	path := os.Getenv("PATH")
	if strings.Contains(strings.ToLower(path), strings.ToLower(dir)) {
		return nil
	}
	// Use PowerShell to add to user PATH via registry
	cmd := exec.Command("powershell", "-Command",
		`[Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";`+dir+`", "User")`)
	return cmd.Run()
}
