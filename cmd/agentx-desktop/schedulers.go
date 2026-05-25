package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Agentx-network/agentx/pkg/config"
)

// SchedulerInfo mirrors the shape returned by the gateway's /api/schedulers
// endpoint. Times are unix milliseconds; the frontend formats them.
type SchedulerInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Message     string `json:"message"`
	Command     string `json:"command,omitempty"`
	Channel     string `json:"channel"`
	ChatID      string `json:"chatId"`
	Kind        string `json:"kind"` // "at" | "every" | "cron"
	AtMS        int64  `json:"atMs,omitempty"`
	EveryMS     int64  `json:"everyMs,omitempty"`
	CronExpr    string `json:"cronExpr,omitempty"`
	CreatedAtMS int64  `json:"createdAtMs"`
}

// SchedulersService is the desktop-side façade for the gateway's schedulers
// API. The dashboard's Schedulers tab uses it to list and cancel scheduled
// cron jobs without going through the chat — the out-of-band escape hatch
// from a runaway reminder/command.
type SchedulersService struct {
	ctx context.Context
}

func NewSchedulersService() *SchedulersService { return &SchedulersService{} }

func (s *SchedulersService) startup(ctx context.Context) { s.ctx = ctx }

// gatewayURL returns the base URL for the running gateway from config.
func (s *SchedulersService) gatewayURL() (string, error) {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:%d", cfg.Gateway.Host, cfg.Gateway.Port), nil
}

// ListSchedulers returns every scheduled cron job currently in the gateway.
func (s *SchedulersService) ListSchedulers() ([]SchedulerInfo, error) {
	base, err := s.gatewayURL()
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(base + "/api/schedulers")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("schedulers endpoint returned %d", resp.StatusCode)
	}
	var out struct {
		Schedulers []SchedulerInfo `json:"schedulers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Schedulers, nil
}

// RemoveScheduler cancels one specific scheduled job by id. Returns 1 if it
// was removed, 0 if it wasn't found (already gone).
func (s *SchedulersService) RemoveScheduler(id string) (int, error) {
	base, err := s.gatewayURL()
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodDelete, base+"/api/schedulers?id="+id, nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("remove returned %d", resp.StatusCode)
	}
	var out struct {
		Removed int `json:"removed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	return out.Removed, nil
}

// RemoveAllSchedulers cancels every scheduled job. The big red button.
// Returns how many were removed.
func (s *SchedulersService) RemoveAllSchedulers() (int, error) {
	base, err := s.gatewayURL()
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodDelete, base+"/api/schedulers/all", nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("remove-all returned %d", resp.StatusCode)
	}
	var out struct {
		Removed int `json:"removed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	return out.Removed, nil
}
