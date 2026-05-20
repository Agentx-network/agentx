package tools

import (
	"context"
	"fmt"
)

type SendCallback func(channel, chatID, content string) error

type MessageTool struct {
	sendCallback   SendCallback
	defaultChannel string
	defaultChatID  string
	sentInRound    bool // Tracks whether a message was sent in the current processing round
}

func NewMessageTool() *MessageTool {
	return &MessageTool{}
}

func (t *MessageTool) Name() string {
	return "message"
}

func (t *MessageTool) Description() string {
	return "Send a proactive notification to the user on a DIFFERENT channel from the one they are currently chatting in. " +
		"Example: user is in desktop chat but you want to send them a Telegram alert when a long-running task finishes. " +
		"Do NOT use this for normal replies in the current conversation — just respond with plain text content for those. " +
		"Sending to the same channel/chat the user is already in is redundant and will be rejected."
}

func (t *MessageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The message content to send",
			},
			"channel": map[string]any{
				"type":        "string",
				"description": "Optional: target channel (telegram, whatsapp, etc.)",
			},
			"chat_id": map[string]any{
				"type":        "string",
				"description": "Optional: target chat/user ID",
			},
		},
		"required": []string{"content"},
	}
}

func (t *MessageTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
	t.sentInRound = false // Reset send tracking for new processing round
}

// HasSentInRound returns true if the message tool sent a message during the current round.
func (t *MessageTool) HasSentInRound() bool {
	return t.sentInRound
}

func (t *MessageTool) SetSendCallback(callback SendCallback) {
	t.sendCallback = callback
}

func (t *MessageTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	content, ok := args["content"].(string)
	if !ok {
		return &ToolResult{ForLLM: "content is required", IsError: true}
	}

	channel, _ := args["channel"].(string)
	chatID, _ := args["chat_id"].(string)

	if channel == "" {
		channel = t.defaultChannel
	}
	if chatID == "" {
		chatID = t.defaultChatID
	}

	if channel == "" || chatID == "" {
		return &ToolResult{ForLLM: "No target channel/chat specified", IsError: true}
	}

	// Reject same-channel sends. Weak LLMs use this tool to "reply" in the
	// current chat, which strands the real content on the outbound bus while
	// the user sees only "Message sent to ...". Tell the model to use plain
	// text instead.
	if tc, ok := GetToolContext(ctx); ok {
		if channel == tc.Channel && chatID == tc.ChatID {
			return &ToolResult{
				ForLLM: "REDUNDANT: You tried to send a message to the SAME channel/chat the user is already in (" +
					channel + ":" + chatID + "). " +
					"To reply in the current conversation, respond with plain text content — do NOT call the message tool. " +
					"The message tool is ONLY for sending notifications to OTHER channels (e.g., user is in desktop but you want to ping their Telegram). " +
					"Try again: either respond as plain text, or specify a different channel.",
				IsError: true,
			}
		}
	}

	if t.sendCallback == nil {
		return &ToolResult{ForLLM: "Message sending not configured", IsError: true}
	}

	if err := t.sendCallback(channel, chatID, content); err != nil {
		return BlockedResult(
			fmt.Sprintf("I couldn't deliver that message to %s. The channel may be offline or misconfigured.", channel),
			fmt.Sprintf("message send to %s:%s failed: %v", channel, chatID, err),
		).WithError(err)
	}

	t.sentInRound = true
	// Silent: user already received the message directly
	return &ToolResult{
		ForLLM: fmt.Sprintf("Message sent to %s:%s", channel, chatID),
		Silent: true,
	}
}
