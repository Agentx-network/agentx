package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMessageTool_Execute_Success(t *testing.T) {
	tool := NewMessageTool()
	tool.SetContext("test-channel", "test-chat-id")

	var sentChannel, sentChatID, sentContent string
	tool.SetSendCallback(func(channel, chatID, content string) error {
		sentChannel = channel
		sentChatID = chatID
		sentContent = content
		return nil
	})

	ctx := context.Background()
	args := map[string]any{
		"content": "Hello, world!",
	}

	result := tool.Execute(ctx, args)

	// Verify message was sent with correct parameters
	if sentChannel != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", sentChannel)
	}
	if sentChatID != "test-chat-id" {
		t.Errorf("Expected chatID 'test-chat-id', got '%s'", sentChatID)
	}
	if sentContent != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", sentContent)
	}

	// Verify ToolResult meets US-011 criteria:
	// - Send success returns SilentResult (Silent=true)
	if !result.Silent {
		t.Error("Expected Silent=true for successful send")
	}

	// - ForLLM contains send status description
	if result.ForLLM != "Message sent to test-channel:test-chat-id" {
		t.Errorf("Expected ForLLM 'Message sent to test-channel:test-chat-id', got '%s'", result.ForLLM)
	}

	// - ForUser is empty (user already received message directly)
	if result.ForUser != "" {
		t.Errorf("Expected ForUser to be empty, got '%s'", result.ForUser)
	}

	// - IsError should be false
	if result.IsError {
		t.Error("Expected IsError=false for successful send")
	}
}

func TestMessageTool_Execute_WithCustomChannel(t *testing.T) {
	tool := NewMessageTool()
	tool.SetContext("default-channel", "default-chat-id")

	var sentChannel, sentChatID string
	tool.SetSendCallback(func(channel, chatID, content string) error {
		sentChannel = channel
		sentChatID = chatID
		return nil
	})

	ctx := context.Background()
	args := map[string]any{
		"content": "Test message",
		"channel": "custom-channel",
		"chat_id": "custom-chat-id",
	}

	result := tool.Execute(ctx, args)

	// Verify custom channel/chatID were used instead of defaults
	if sentChannel != "custom-channel" {
		t.Errorf("Expected channel 'custom-channel', got '%s'", sentChannel)
	}
	if sentChatID != "custom-chat-id" {
		t.Errorf("Expected chatID 'custom-chat-id', got '%s'", sentChatID)
	}

	if !result.Silent {
		t.Error("Expected Silent=true")
	}
	if result.ForLLM != "Message sent to custom-channel:custom-chat-id" {
		t.Errorf("Expected ForLLM 'Message sent to custom-channel:custom-chat-id', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_SendFailure(t *testing.T) {
	tool := NewMessageTool()
	tool.SetContext("test-channel", "test-chat-id")

	sendErr := errors.New("network error")
	tool.SetSendCallback(func(channel, chatID, content string) error {
		return sendErr
	})

	ctx := context.Background()
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify ToolResult for send failure (now a BlockedResult: clean ForUser,
	// technical ForLLM, original Err preserved):
	if !result.IsError {
		t.Error("Expected IsError=true for failed send")
	}

	// - ForLLM keeps the technical detail for the model
	if !strings.Contains(result.ForLLM, "network error") {
		t.Errorf("Expected ForLLM to contain 'network error', got '%s'", result.ForLLM)
	}

	// - ForUser is a clean, non-technical message
	if result.ForUser == "" || strings.Contains(result.ForUser, "network error") {
		t.Errorf("Expected a clean user-facing message without raw error, got '%s'", result.ForUser)
	}

	// - Err field should contain original error
	if result.Err != sendErr {
		t.Errorf("Expected Err to be sendErr, got %v", result.Err)
	}
}

func TestMessageTool_Execute_MissingContent(t *testing.T) {
	tool := NewMessageTool()
	tool.SetContext("test-channel", "test-chat-id")

	ctx := context.Background()
	args := map[string]any{} // content missing

	result := tool.Execute(ctx, args)

	// Verify error result for missing content
	if !result.IsError {
		t.Error("Expected IsError=true for missing content")
	}
	if result.ForLLM != "content is required" {
		t.Errorf("Expected ForLLM 'content is required', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_NoTargetChannel(t *testing.T) {
	tool := NewMessageTool()
	// No SetContext called, so defaultChannel and defaultChatID are empty

	tool.SetSendCallback(func(channel, chatID, content string) error {
		return nil
	})

	ctx := context.Background()
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify error when no target channel specified
	if !result.IsError {
		t.Error("Expected IsError=true when no target channel")
	}
	if result.ForLLM != "No target channel/chat specified" {
		t.Errorf("Expected ForLLM 'No target channel/chat specified', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_NotConfigured(t *testing.T) {
	tool := NewMessageTool()
	tool.SetContext("test-channel", "test-chat-id")
	// No SetSendCallback called

	ctx := context.Background()
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify error when send callback not configured
	if !result.IsError {
		t.Error("Expected IsError=true when send callback not configured")
	}
	if result.ForLLM != "Message sending not configured" {
		t.Errorf("Expected ForLLM 'Message sending not configured', got '%s'", result.ForLLM)
	}
}

// In the desktop chat the model often calls message without a channel, so the
// target would default back to "desktop" and be rejected as redundant. With a
// push-target resolver, an empty/same-channel target redirects to the connected
// push channel (Telegram) instead of bouncing.
func TestMessageTool_Execute_RedirectsToPushChannel(t *testing.T) {
	tool := NewMessageTool()
	tool.SetContext("desktop", "chat")
	tool.SetPushTargetResolver(func() (string, string, bool) {
		return "telegram", "1046193410", true
	})

	var sentChannel, sentChatID string
	tool.SetSendCallback(func(channel, chatID, content string) error {
		sentChannel, sentChatID = channel, chatID
		return nil
	})

	// Simulate the user currently chatting in desktop.
	ctx := WithToolContext(context.Background(), ToolContext{Channel: "desktop", ChatID: "chat"})
	result := tool.Execute(ctx, map[string]any{"content": "ping!"})

	if result.IsError {
		t.Fatalf("expected redirect to succeed, got error: %s", result.ForLLM)
	}
	if sentChannel != "telegram" || sentChatID != "1046193410" {
		t.Errorf("expected redirect to telegram:1046193410, got %s:%s", sentChannel, sentChatID)
	}
}

// When there is no push channel to redirect to and the target is the channel the
// user is already in, the tool must reject as REDUNDANT (use plain text).
func TestMessageTool_Execute_RedundantWhenNoPushChannel(t *testing.T) {
	tool := NewMessageTool()
	tool.SetContext("desktop", "chat")
	tool.SetPushTargetResolver(func() (string, string, bool) { return "", "", false })
	tool.SetSendCallback(func(channel, chatID, content string) error { return nil })

	ctx := WithToolContext(context.Background(), ToolContext{Channel: "desktop", ChatID: "chat"})
	result := tool.Execute(ctx, map[string]any{"content": "hello"})

	if !result.IsError {
		t.Error("expected REDUNDANT error when target is the current channel and no push channel exists")
	}
	if !strings.Contains(result.ForLLM, "REDUNDANT") {
		t.Errorf("expected REDUNDANT message, got: %s", result.ForLLM)
	}
}

func TestMessageTool_Name(t *testing.T) {
	tool := NewMessageTool()
	if tool.Name() != "message" {
		t.Errorf("Expected name 'message', got '%s'", tool.Name())
	}
}

func TestMessageTool_Description(t *testing.T) {
	tool := NewMessageTool()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestMessageTool_Parameters(t *testing.T) {
	tool := NewMessageTool()
	params := tool.Parameters()

	// Verify parameters structure
	typ, ok := params["type"].(string)
	if !ok || typ != "object" {
		t.Error("Expected type 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check required properties
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "content" {
		t.Error("Expected 'content' to be required")
	}

	// Check content property
	contentProp, ok := props["content"].(map[string]any)
	if !ok {
		t.Error("Expected 'content' property")
	}
	if contentProp["type"] != "string" {
		t.Error("Expected content type to be 'string'")
	}

	// Check channel property (optional)
	channelProp, ok := props["channel"].(map[string]any)
	if !ok {
		t.Error("Expected 'channel' property")
	}
	if channelProp["type"] != "string" {
		t.Error("Expected channel type to be 'string'")
	}

	// Check chat_id property (optional)
	chatIDProp, ok := props["chat_id"].(map[string]any)
	if !ok {
		t.Error("Expected 'chat_id' property")
	}
	if chatIDProp["type"] != "string" {
		t.Error("Expected chat_id type to be 'string'")
	}
}
