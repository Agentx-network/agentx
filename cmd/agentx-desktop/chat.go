package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Agentx-network/agentx/pkg/config"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type ChatRequest struct {
	Message    string `json:"message"`
	SessionKey string `json:"sessionKey"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

type ChatService struct {
	ctx context.Context
}

func NewChatService() *ChatService {
	return &ChatService{}
}

func (c *ChatService) startup(ctx context.Context) {
	c.ctx = ctx
}

// SendMessage sends a message to the gateway's SSE chat endpoint,
// emits "chat:delta" Wails events as tokens arrive, and returns the
// final response.
func (c *ChatService) SendMessage(message string, sessionKey string) (*ChatResponse, error) {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if sessionKey == "" {
		sessionKey = "desktop:chat"
	}

	chatURL := fmt.Sprintf("http://%s:%d/api/chat", cfg.Gateway.Host, cfg.Gateway.Port)

	reqBody, err := json.Marshal(ChatRequest{
		Message:    message,
		SessionKey: sessionKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(chatURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("gateway not reachable: %w", err)
	}
	defer resp.Body.Close()

	// Read SSE stream
	scanner := bufio.NewScanner(resp.Body)
	var finalResponse string

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "delta":
			delta, _ := event["delta"].(string)
			if delta != "" {
				wailsRuntime.EventsEmit(c.ctx, "chat:delta", delta)
			}
		case "done":
			finalResponse, _ = event["response"].(string)
			wailsRuntime.EventsEmit(c.ctx, "chat:done", finalResponse)
		case "error":
			errMsg, _ := event["error"].(string)
			return nil, fmt.Errorf("gateway error: %s", errMsg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream read error: %w", err)
	}

	return &ChatResponse{Response: finalResponse}, nil
}

// IsGatewayReachable checks if the gateway is available.
func (c *ChatService) IsGatewayReachable() bool {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return false
	}

	healthURL := fmt.Sprintf("http://%s:%d/health", cfg.Gateway.Host, cfg.Gateway.Port)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
