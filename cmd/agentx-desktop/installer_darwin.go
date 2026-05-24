//go:build darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func getInstallDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}

func updatePATH(dir string) error {
	path := os.Getenv("PATH")
	if strings.Contains(path, dir) {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	line := fmt.Sprintf("\nexport PATH=\"%s:$PATH\"\n", dir)
	for _, rc := range []string{".zshrc", ".bash_profile", ".profile"} {
		rcPath := filepath.Join(home, rc)
		if _, err := os.Stat(rcPath); err == nil {
			data, _ := os.ReadFile(rcPath)
			if !strings.Contains(string(data), dir) {
				f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_WRONLY, 0o644)
				if err == nil {
					f.WriteString(line)
					f.Close()
				}
			}
		}
	}
	return nil
}
