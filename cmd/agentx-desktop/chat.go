package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Agentx-network/agentx/pkg/attach"
	"github.com/Agentx-network/agentx/pkg/config"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type ChatRequest struct {
	Message    string   `json:"message"`
	SessionKey string   `json:"sessionKey"`
	Media      []string `json:"media,omitempty"`
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
func (c *ChatService) SendMessage(message string, sessionKey string, media []string) (*ChatResponse, error) {
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
		Media:      media,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// No overall timeout — the SSE stream can run for several minutes
	// while the agent processes tool calls. We rely on the server to
	// close the connection when done (or the Wails context cancelling).
	// ResponseHeaderTimeout is set to 5 minutes so a slow first LLM
	// call (rate-limited free tier, cold start, large session context,
	// retry-after windows) doesn't cause a misleading "gateway not
	// reachable" error in the UI while the gateway is in fact working.
	// The previous 30s ceiling fired routinely on free-tier providers.
	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 5 * time.Minute,
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
		return readHistoryForSession(filepath.Join(sessionsDir, filename))
	}

	// Otherwise load the desktop's own conversation. The desktop chat persists
	// as the agent MAIN session ("agent:<id>:main" → "agent_..._main.json"),
	// whereas channel DMs (Telegram, Discord, …) persist as scoped sessions
	// ("agent:<id>:telegram:direct:<peer>"). Previously we returned the most
	// recently-updated session of ANY kind, so chatting on Telegram made its
	// history show up in the desktop. Restrict to "*_main.json" so the desktop
	// only ever shows the desktop/main conversation.
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_main.json") {
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

	return readHistoryForSession(bestPath)
}

// readHistoryForSession returns the display history for a session, preferring
// the append-only transcript (<key>.transcript.jsonl) — which is never
// summarized, so it holds the full conversation — and falling back to the
// session .json (summarized/truncated) for sessions created before transcripts
// existed.
func readHistoryForSession(sessionJSONPath string) ([]HistoryMessage, error) {
	transcriptPath := strings.TrimSuffix(sessionJSONPath, ".json") + ".transcript.jsonl"
	if msgs, err := readTranscriptFile(transcriptPath); err == nil && len(msgs) > 0 {
		return msgs, nil
	}
	return readSessionFile(sessionJSONPath)
}

// readTranscriptFile reads an append-only JSONL transcript into display history.
// Malformed lines are skipped rather than failing the whole load.
func readTranscriptFile(path string) ([]HistoryMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var history []HistoryMessage
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			Timestamp int64  `json:"ts"`
		}
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if (e.Role != "user" && e.Role != "assistant") || e.Content == "" {
			continue
		}
		history = append(history, HistoryMessage{
			Role:      e.Role,
			Content:   e.Content,
			Timestamp: e.Timestamp,
		})
	}
	return history, nil
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

	// Rate limit — checked BEFORE billing because Groq's 413 message
	// includes a URL to /settings/billing in their upgrade hint, which
	// previously false-matched the billing branch below. Catch the
	// telltale rate-limit wordings explicitly (Groq's 413, OpenAI's 429,
	// generic "too many requests", quota mentions).
	if strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "rate_limit") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(lower, "429") ||
		strings.Contains(lower, "request too large") ||
		strings.Contains(lower, "tokens per minute") ||
		strings.Contains(lower, "reduce your message size") ||
		strings.Contains(lower, "tpm") ||
		strings.Contains(lower, "quota") {
		return "Rate limited by the provider. Please wait a moment and try again."
	}

	// Billing — only match phrases that unambiguously indicate a billing
	// problem. Bare "billing" used to false-match Groq's upgrade-link URL
	// in rate-limit error bodies (Bug #29).
	if strings.Contains(lower, "402") ||
		strings.Contains(lower, "payment required") ||
		strings.Contains(lower, "insufficient credits") ||
		strings.Contains(lower, "insufficient balance") ||
		strings.Contains(lower, "billing problem") ||
		strings.Contains(lower, "billing required") ||
		strings.Contains(lower, "plans & billing") {
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
// ReadImageDataURL reads a generated image from disk and returns it as a
// base64 data URL the webview can render inline. The chat detects an
// "IMAGE:<path>" marker (emitted by the image_generate tool) and calls this to
// display the result. Only image files under a few MB are served.
func (c *ChatService) ReadImageDataURL(path string) (string, error) {
	path = normalizeImagePath(path)
	if path == "" {
		return "", fmt.Errorf("image path is required")
	}
	mime := ""
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		mime = "image/png"
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	default:
		return "", fmt.Errorf("unsupported image type: %s", filepath.Ext(path))
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("image not found: %w", err)
	}
	if info.Size() > 16<<20 {
		return "", fmt.Errorf("image too large to display (%d bytes)", info.Size())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// normalizeImagePath trims a path that may arrive wrapped by the model: a
// "file://" URL, surrounding quotes/whitespace, or markdown angle brackets.
func normalizeImagePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "<>\"'")
	path = strings.TrimPrefix(path, "file://")
	return strings.TrimSpace(path)
}

// SaveImageAs lets the user download a generated image: it opens a native
// Save-As dialog and copies the file to the chosen destination. Returns the
// saved path, or "" if the user cancelled.
func (c *ChatService) SaveImageAs(srcPath string) (string, error) {
	srcPath = normalizeImagePath(srcPath)
	if srcPath == "" {
		return "", fmt.Errorf("no image path")
	}
	if _, err := os.Stat(srcPath); err != nil {
		return "", fmt.Errorf("image not found: %w", err)
	}
	dest, err := wailsRuntime.SaveFileDialog(c.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Save image",
		DefaultFilename: filepath.Base(srcPath),
	})
	if err != nil {
		return "", err
	}
	if dest == "" {
		return "", nil // user cancelled
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", fmt.Errorf("save image: %w", err)
	}
	return dest, nil
}

// Attachment describes a validated, stored upload returned to the frontend.
type Attachment struct {
	Path string `json:"path"` // absolute path under <workspace>/uploads
	Name string `json:"name"`
	Size int64  `json:"size"`
	Kind string `json:"kind"` // image | audio | doc
}

// RejectedAttachment describes a file the user picked that was not accepted.
type RejectedAttachment struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// PickAttachmentsResult is returned by PickAttachments.
type PickAttachmentsResult struct {
	Accepted []Attachment         `json:"accepted"`
	Rejected []RejectedAttachment `json:"rejected"`
}

// PickAttachments opens a native multi-file picker, validates each selection,
// and copies the accepted ones into <workspace>/uploads/. Returns accepted
// attachments (with their stored paths) plus any rejections with reasons.
// Enforces the per-message file-count and total-size limits so the user gets
// immediate feedback before sending.
func (c *ChatService) PickAttachments() (*PickAttachmentsResult, error) {
	paths, err := wailsRuntime.OpenMultipleFilesDialog(c.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Attach files",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "Supported files (images, audio, video, documents)",
				Pattern:     "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.mp3;*.wav;*.m4a;*.ogg;*.flac;*.aac;*.mp4;*.mov;*.webm;*.mkv;*.avi;*.m4v;*.flv;*.wmv;*.mpeg;*.mpg;*.pdf;*.txt;*.md;*.markdown;*.csv;*.json;*.log;*.yaml;*.yml;*.xml;*.html;*.htm;*.tsv;*.ini;*.toml",
			},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, err
	}

	result := &PickAttachmentsResult{Accepted: []Attachment{}, Rejected: []RejectedAttachment{}}
	if len(paths) == 0 {
		return result, nil // user canceled
	}

	cfg, err := config.LoadConfig(getConfigPath())
	workspace := filepath.Join(os.Getenv("HOME"), ".agentx", "workspace")
	if err == nil {
		workspace = cfg.WorkspacePath()
	}

	if len(paths) > attach.MaxFiles {
		return nil, fmt.Errorf("you can attach at most %d files at a time", attach.MaxFiles)
	}

	var total int64
	for _, p := range paths {
		stored, verr := attach.ValidateAndStore(p, workspace)
		if verr != nil {
			result.Rejected = append(result.Rejected, RejectedAttachment{
				Name:   filepath.Base(p),
				Reason: verr.Error(),
			})
			continue
		}
		total += stored.Size
		if total > attach.MaxTotalBytes {
			result.Rejected = append(result.Rejected, RejectedAttachment{
				Name:   stored.Name,
				Reason: "skipped — total attachment size limit reached",
			})
			_ = os.Remove(stored.Path)
			continue
		}
		result.Accepted = append(result.Accepted, Attachment{
			Path: stored.Path, Name: stored.Name, Size: stored.Size, Kind: string(stored.Kind),
		})
	}
	return result, nil
}

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

// PollNotifications drains any proactively-delivered messages for the desktop
// chat (cron reminders that fired while the user wasn't mid-request). The chat
// page calls this on a short timer and renders each as an assistant message.
func (c *ChatService) PollNotifications() ([]string, error) {
	cfg, err := config.LoadConfig(getConfigPath())
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("http://%s:%d/api/notifications?chatID=chat", cfg.Gateway.Host, cfg.Gateway.Port)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("notifications endpoint returned %d", resp.StatusCode)
	}
	var out struct {
		Messages []string `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Messages, nil
}
