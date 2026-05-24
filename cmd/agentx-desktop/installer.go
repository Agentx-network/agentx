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
		versionCmd := exec.Command(binPath, "version")
		hideConsoleWindow(versionCmd)
		if out, err := versionCmd.Output(); err == nil {
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
		Timeout: 15 * time.Second,
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

	// Resolve the latest tag first, then use the direct download URL
	// (avoids the extra /latest redirect which can timeout on some networks)
	tag, tagErr := s.GetLatestRelease()
	if tagErr != nil || tag == "" {
		tag = "latest"
	}
	var url string
	if tag == "latest" {
		url = fmt.Sprintf("https://github.com/Agentx-network/agentx/releases/latest/download/%s", assetName)
	} else {
		url = fmt.Sprintf("https://github.com/Agentx-network/agentx/releases/download/%s/%s", tag, assetName)
	}

	if err := os.MkdirAll(platform.InstallDir, 0o755); err != nil {
		return fmt.Errorf("cannot create install directory: %w", err)
	}

	tmpFile := platform.BinaryPath + ".tmp"

	// Resumable download — GitHub CDN can stall mid-transfer.
	// We retry up to 5 times, using Range headers to resume from where we left off.
	const maxAttempts = 5
	var totalBytes int64
	var downloaded int64

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, reqErr := http.NewRequest("GET", url, nil)
		if reqErr != nil {
			return fmt.Errorf("failed to create request: %w", reqErr)
		}

		// Resume from where we left off
		if downloaded > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", downloaded))
		}

		dlClient := &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: 30 * time.Second,
			},
		}
		resp, err := dlClient.Do(req)
		if err != nil {
			if attempt < maxAttempts {
				wailsRuntime.EventsEmit(s.ctx, "download:progress", map[string]interface{}{
					"status": fmt.Sprintf("Connection failed, retry %d/%d...", attempt, maxAttempts),
				})
				time.Sleep(time.Duration(attempt*3) * time.Second)
				continue
			}
			return fmt.Errorf("download failed after %d attempts: %w", maxAttempts, err)
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			if attempt < maxAttempts {
				time.Sleep(time.Duration(attempt*3) * time.Second)
				continue
			}
			return fmt.Errorf("download failed with status %d for %s", resp.StatusCode, assetName)
		}

		// Get total size from first response
		if totalBytes == 0 {
			if resp.StatusCode == http.StatusOK {
				totalBytes = resp.ContentLength
			} else if resp.StatusCode == http.StatusPartialContent {
				// Parse Content-Range: bytes 1234-5678/9012
				cr := resp.Header.Get("Content-Range")
				if cr != "" {
					fmt.Sscanf(cr, "bytes %*d-%*d/%d", &totalBytes)
				}
			}
		}

		// Open file for append (or create)
		var f *os.File
		if downloaded > 0 {
			f, err = os.OpenFile(tmpFile, os.O_WRONLY|os.O_APPEND, 0o644)
		} else {
			f, err = os.Create(tmpFile)
		}
		if err != nil {
			resp.Body.Close()
			return fmt.Errorf("cannot open temp file: %w", err)
		}

		// Read with a per-chunk deadline to detect stalls
		buf := make([]byte, 64*1024)
		stalled := false
		for {
			// Set a 30s read deadline per chunk — if nothing arrives, it's stalled
			type deadliner interface{ SetReadDeadline(time.Time) error }
			if dl, ok := resp.Body.(deadliner); ok {
				dl.SetReadDeadline(time.Now().Add(30 * time.Second))
			}

			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := f.Write(buf[:n]); writeErr != nil {
					f.Close()
					resp.Body.Close()
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
				stalled = true
				break
			}
		}
		f.Close()
		resp.Body.Close()

		if !stalled {
			break // download complete
		}

		if attempt < maxAttempts {
			wailsRuntime.EventsEmit(s.ctx, "download:progress", map[string]interface{}{
				"status":     fmt.Sprintf("Download stalled at %d%%, resuming... (%d/%d)", int(float64(downloaded)/float64(totalBytes)*100), attempt, maxAttempts),
				"downloaded": downloaded,
				"total":      totalBytes,
			})
			time.Sleep(time.Duration(attempt*3) * time.Second)
		} else {
			os.Remove(tmpFile)
			return fmt.Errorf("download stalled after %d attempts (got %d/%d bytes)", maxAttempts, downloaded, totalBytes)
		}
	}

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
