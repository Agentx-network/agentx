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

	"regexp"

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

	// No overall timeout — the SSE stream can run for several minutes
	// while the agent processes tool calls. We rely on the server to
	// close the connection when done (or the Wails context cancelling).
	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second, // wait up to 30s for initial response headers
		},
	}
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
			return nil, fmt.Errorf("%s", friendlyError(errMsg))
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

// friendlyError converts raw provider/gateway error messages into
// clean, user-facing English messages for the chat UI.
func friendlyError(raw string) string {
	lower := strings.ToLower(raw)

	// Auth / API key errors
	if strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "401") ||
		strings.Contains(lower, "invalid api key") ||
		strings.Contains(lower, "invalid_api_key") ||
		strings.Contains(lower, "incorrect api key") ||
		strings.Contains(lower, "authentication") ||
		strings.Contains(lower, "no api key") ||
		strings.Contains(lower, "no credentials") {
		return "Authentication failed — check your API key in Config > Models."
	}

	// Rate limit
	if strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "rate_limit") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(lower, "429") ||
		strings.Contains(lower, "quota") {
		return "Rate limited by the provider. Please wait a moment and try again."
	}

	// Billing
	if strings.Contains(lower, "402") ||
		strings.Contains(lower, "payment required") ||
		strings.Contains(lower, "insufficient credits") ||
		strings.Contains(lower, "insufficient balance") ||
		strings.Contains(lower, "billing") {
		return "Billing issue — your provider account may need credits or a payment method."
	}

	// Overloaded
	if strings.Contains(lower, "overloaded") {
		return "The AI provider is currently overloaded. Please try again in a moment."
	}

	// Timeout / connectivity
	if strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "timed out") ||
		strings.Contains(lower, "deadline exceeded") {
		return "Request timed out. The provider may be slow — try again."
	}

	// Connection errors
	if strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "dns") {
		return "Cannot reach the AI provider. Check your internet connection and API base URL."
	}

	// Strip non-ASCII (e.g., Chinese/Japanese error text) and return a cleaned version.
	cleaned := regexp.MustCompile(`[^\x20-\x7E]`).ReplaceAllString(raw, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" || len(cleaned) < 10 {
		return "An unexpected error occurred. Check your provider settings in Config."
	}

	return cleaned
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
