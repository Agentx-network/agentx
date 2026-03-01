package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type PlatformInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	InstallDir   string `json:"installDir"`
	BinaryPath   string `json:"binaryPath"`
	BinaryExists bool   `json:"binaryExists"`
	Version      string `json:"version"`
}

type InstallerService struct {
	ctx context.Context
}

func NewInstallerService() *InstallerService {
	return &InstallerService{}
}

func (s *InstallerService) startup(ctx context.Context) {
	s.ctx = ctx
}

func mapArch(goarch string) string {
	switch goarch {
	case "arm":
		return "armv7"
	default:
		return goarch
	}
}

func (s *InstallerService) DetectPlatform() PlatformInfo {
	installDir := getInstallDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	// Check install dir first, then fall back to findBinary() (PATH + standard locations).
	binPath := filepath.Join(installDir, "agentx"+ext)
	exists := false
	version := ""
	if _, err := os.Stat(binPath); err == nil {
		exists = true
	} else if resolved, err := findBinary(); err == nil {
		binPath = resolved
		exists = true
	}

	if exists {
		if out, err := exec.Command(binPath, "version").Output(); err == nil {
			version = strings.TrimSpace(string(out))
		}
	}

	return PlatformInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		InstallDir:   installDir,
		BinaryPath:   binPath,
		BinaryExists: exists,
		Version:      version,
	}
}

func (s *InstallerService) GetLatestRelease() (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("https://github.com/Agentx-network/agentx/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("could not determine latest release")
	}
	parts := strings.Split(loc, "/")
	return parts[len(parts)-1], nil
}

func (s *InstallerService) InstallBinary() error {
	platform := s.DetectPlatform()
	osName := platform.OS
	archName := mapArch(platform.Arch)

	ext := ""
	if osName == "windows" {
		ext = ".exe"
	}

	// Raw binary download — matches release asset naming: agentx-{os}-{arch}[.exe]
	assetName := fmt.Sprintf("agentx-%s-%s%s", osName, archName, ext)
	url := fmt.Sprintf("https://github.com/Agentx-network/agentx/releases/latest/download/%s", assetName)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d for %s", resp.StatusCode, assetName)
	}

	if err := os.MkdirAll(platform.InstallDir, 0o755); err != nil {
		return fmt.Errorf("cannot create install directory: %w", err)
	}

	tmpFile := platform.BinaryPath + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}

	totalBytes := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				f.Close()
				os.Remove(tmpFile)
				return writeErr
			}
			downloaded += int64(n)
			if totalBytes > 0 {
				pct := float64(downloaded) / float64(totalBytes) * 100
				wailsRuntime.EventsEmit(s.ctx, "download:progress", map[string]interface{}{
					"downloaded": downloaded,
					"total":      totalBytes,
					"percent":    pct,
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			f.Close()
			os.Remove(tmpFile)
			return readErr
		}
	}
	f.Close()

	if err := os.Chmod(tmpFile, 0o755); err != nil {
		os.Remove(tmpFile)
		return err
	}

	// Stop gateway and kill processes before replacing the binary —
	// on Windows the running exe has a file lock that blocks rename.
	if platform.BinaryExists {
		s.UninstallService()
		time.Sleep(2 * time.Second)
	}

	// On Windows, a recently-killed process may still hold a file lock.
	// Rename the old binary out of the way first (Windows allows renaming
	// a running/locked exe), then rename the new one in, with retries.
	if runtime.GOOS == "windows" && platform.BinaryExists {
		oldPath := platform.BinaryPath + ".old"
		os.Remove(oldPath) // clean up any previous .old file
		// Move locked binary aside — this usually succeeds even if the file is locked.
		_ = os.Rename(platform.BinaryPath, oldPath)
		// Best-effort cleanup; may fail if still locked — that's fine.
		defer os.Remove(oldPath)
	}

	var renameErr error
	for i := 0; i < 5; i++ {
		if renameErr = os.Rename(tmpFile, platform.BinaryPath); renameErr == nil {
			break
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if renameErr != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("install failed: %w", renameErr)
	}

	wailsRuntime.EventsEmit(s.ctx, "download:progress", map[string]interface{}{
		"downloaded": totalBytes,
		"total":      totalBytes,
		"percent":    100.0,
	})

	return updatePATH(platform.InstallDir)
}
