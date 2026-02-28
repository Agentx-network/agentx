package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/health"
)

type GatewayStatus struct {
	Running  bool                   `json:"running"`
	Health   *health.StatusResponse `json:"health,omitempty"`
	Channels []ChannelInfo          `json:"channels"`
	Models   []ModelInfo            `json:"models"`
}

type ChannelInfo struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type ModelInfo struct {
	ModelName string `json:"modelName"`
	Model     string `json:"model"`
	HasKey    bool   `json:"hasKey"`
}

type DashboardService struct {
	ctx context.Context
}

func NewDashboardService() *DashboardService {
	return &DashboardService{}
}

func (d *DashboardService) startup(ctx context.Context) {
	d.ctx = ctx
}

func (d *DashboardService) GetStatus() GatewayStatus {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return GatewayStatus{}
	}

	status := GatewayStatus{
		Channels: getChannelInfos(cfg),
		Models:   getModelInfos(cfg),
	}

	healthURL := fmt.Sprintf("http://%s:%d/health", cfg.Gateway.Host, cfg.Gateway.Port)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		status.Running = false
		return status
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		status.Running = false
		return status
	}

	var sr health.StatusResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		status.Running = false
		return status
	}

	status.Running = true
	status.Health = &sr
	return status
}

func (d *DashboardService) GetLogs(lines int) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	logPath := filepath.Join(home, ".agentx", "gateway.log")
	if _, err := os.Stat(logPath); err == nil {
		data, err := os.ReadFile(logPath)
		if err != nil {
			return "", err
		}
		allLines := strings.Split(string(data), "\n")
		if len(allLines) > lines {
			allLines = allLines[len(allLines)-lines:]
		}
		return strings.Join(allLines, "\n"), nil
	}

	// Try journalctl on Linux
	if runtime.GOOS == "linux" {
		cmd := exec.Command("journalctl", "--user", "-u", "agentx-gateway.service",
			"-n", fmt.Sprintf("%d", lines), "--no-pager")
		out, err := cmd.Output()
		if err == nil {
			return string(out), nil
		}
	}

	return "", fmt.Errorf("no logs found")
}

func (d *DashboardService) StartGateway() error {
	binPath, err := findBinary()
	if err != nil {
		return err
	}

	if runtime.GOOS == "linux" {
		if out, err := exec.Command("systemctl", "--user", "start", "agentx-gateway.service").CombinedOutput(); err != nil {
			// Fallback: run directly
			cmd := exec.Command(binPath, "gateway")
			cmd.Stdout = nil
			cmd.Stderr = nil
			return cmd.Start()
		} else {
			_ = out
			return nil
		}
	}
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("launchctl", "start", "com.agentx.gateway").CombinedOutput(); err != nil {
			cmd := exec.Command(binPath, "gateway")
			cmd.Stdout = nil
			cmd.Stderr = nil
			return cmd.Start()
		} else {
			_ = out
			return nil
		}
	}

	cmd := exec.Command(binPath, "gateway")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func (d *DashboardService) StopGateway() error {
	// Try service manager first
	if runtime.GOOS == "linux" {
		exec.Command("systemctl", "--user", "stop", "agentx-gateway.service").Run()
	}
	if runtime.GOOS == "darwin" {
		exec.Command("launchctl", "stop", "com.agentx.gateway").Run()
	}

	// Always kill any remaining agentx gateway processes directly
	killGatewayProcesses()
	return nil
}

func (d *DashboardService) RestartGateway() error {
	if err := d.StopGateway(); err != nil {
		// Ignore stop errors, gateway might not be running
	}
	time.Sleep(500 * time.Millisecond)
	return d.StartGateway()
}

func findBinary() (string, error) {
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

func getChannelInfos(cfg *config.Config) []ChannelInfo {
	return []ChannelInfo{
		{Name: "Telegram", Enabled: cfg.Channels.Telegram.Enabled},
		{Name: "Discord", Enabled: cfg.Channels.Discord.Enabled},
		{Name: "Slack", Enabled: cfg.Channels.Slack.Enabled},
		{Name: "WhatsApp", Enabled: cfg.Channels.WhatsApp.Enabled},
		{Name: "Feishu", Enabled: cfg.Channels.Feishu.Enabled},
		{Name: "DingTalk", Enabled: cfg.Channels.DingTalk.Enabled},
		{Name: "QQ", Enabled: cfg.Channels.QQ.Enabled},
		{Name: "LINE", Enabled: cfg.Channels.LINE.Enabled},
		{Name: "OneBot", Enabled: cfg.Channels.OneBot.Enabled},
		{Name: "WeCom", Enabled: cfg.Channels.WeCom.Enabled},
		{Name: "WeComApp", Enabled: cfg.Channels.WeComApp.Enabled},
		{Name: "MaixCam", Enabled: cfg.Channels.MaixCam.Enabled},
	}
}

func getModelInfos(cfg *config.Config) []ModelInfo {
	var models []ModelInfo
	for _, m := range cfg.ModelList {
		models = append(models, ModelInfo{
			ModelName: m.ModelName,
			Model:     m.Model,
			HasKey:    m.APIKey != "",
		})
	}
	return models
}
