package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

// HistoryMessage is a simplified message returned to the frontend.
type HistoryMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"` // unix millis
}

// GetChatHistory reads saved session history from disk and returns
// user/assistant messages for display in the frontend.
// It scans all session files and returns the most recently updated one,
// or uses the provided sessionKey to find a specific session.
func (c *ChatService) GetChatHistory(sessionKey string) ([]HistoryMessage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	sessionsDir := filepath.Join(home, ".agentx", "workspace", "sessions")

	// If a specific key is given, look for that file directly
	if sessionKey != "" {
		filename := strings.ReplaceAll(sessionKey, ":", "_") + ".json"
		return readSessionFile(filepath.Join(sessionsDir, filename))
	}

	// Otherwise find the most recently updated session file
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryMessage{}, nil
		}
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var bestPath string
	var bestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(bestTime) {
			bestTime = info.ModTime()
			bestPath = filepath.Join(sessionsDir, entry.Name())
		}
	}

	if bestPath == "" {
		return []HistoryMessage{}, nil
	}

	return readSessionFile(bestPath)
}

func readSessionFile(path string) ([]HistoryMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryMessage{}, nil
		}
		return nil, fmt.Errorf("read session file: %w", err)
	}

	var session struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Updated time.Time `json:"updated"`
	}
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse session file: %w", err)
	}

	var history []HistoryMessage
	for _, msg := range session.Messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}
		if msg.Content == "" {
			continue
		}
		history = append(history, HistoryMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: session.Updated.UnixMilli(),
		})
	}

	return history, nil
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
