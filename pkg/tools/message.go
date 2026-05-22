package tools

import (
	"context"
	"fmt"
	"strings"
)

type SendCallback func(channel, chatID, content string) error

// PushTargetResolver returns the preferred proactive-notification target — a
// connected push channel (e.g. telegram) and the owner's chat ID — or ok=false
// when no push channel is connected. Lives in the agent layer because it needs
// live channel config.
type PushTargetResolver func() (channel, chatID string, ok bool)

type MessageTool struct {
	sendCallback   SendCallback
	pushTarget     PushTargetResolver
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

// SetPushTargetResolver wires the resolver used to redirect a proactive message
// to a connected push channel when the model didn't name a usable one.
func (t *MessageTool) SetPushTargetResolver(r PushTargetResolver) {
	t.pushTarget = r
}

func (t *MessageTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	content, _ := args["content"].(string)
	if strings.TrimSpace(content) == "" {
		return &ToolResult{ForLLM: "content is required", IsError: true}
	}

	channel := strings.TrimSpace(asString(args["channel"]))
	chatID := strings.TrimSpace(asString(args["chat_id"]))

	if channel == "" {
		channel = t.defaultChannel
	}
	if chatID == "" {
		chatID = t.defaultChatID
	}

	// Where the user currently is (used to detect a redundant same-channel send).
	curChannel, curChatID := "", ""
	if tc, ok := GetToolContext(ctx); ok {
		curChannel, curChatID = tc.Channel, tc.ChatID
	}

	// The message tool exists to notify the user on a PUSH channel (Telegram,
	// etc.) — typically while they're chatting somewhere that can't receive a
	// proactive push (desktop/CLI). When the resolved target is empty, or is the
	// very channel the user is already in, redirect to a connected push channel
	// instead of bouncing. This is the common "ping me on Telegram from the
	// desktop app" case, where the model omits the channel and it would
	// otherwise default back to desktop and be rejected as redundant.
	redundant := channel == "" || (curChannel != "" && channel == curChannel && chatID == curChatID)
	if redundant && t.pushTarget != nil {
		if pc, pcid, ok := t.pushTarget(); ok && pc != curChannel {
			channel, chatID, redundant = pc, pcid, false
		}
	}

	if channel == "" || chatID == "" {
		if curChannel != "" {
			return &ToolResult{
				ForLLM: "REDUNDANT: there is no separate channel to notify — the user is already in \"" + curChannel + "\". " +
					"To reply in the current conversation, respond with plain text content; do NOT call the message tool. " +
					"The message tool is ONLY for reaching the user on a DIFFERENT, connected push channel (e.g. Telegram). " +
					"If no push channel is connected, tell the user to connect one in Config → Channels.",
				IsError: true,
			}
		}
		return &ToolResult{ForLLM: "No target channel/chat specified", IsError: true}
	}

	// Target is the same channel/chat the user is already in, and there was no
	// push channel to redirect to. Weak LLMs use the message tool to "reply" in
	// the current chat, which strands the content on the outbound bus while the
	// user sees only "Message sent to ...". Tell the model to use plain text.
	if redundant {
		return &ToolResult{
			ForLLM: "REDUNDANT: You tried to send a message to the SAME channel/chat the user is already in (" +
				channel + ":" + chatID + "). " +
				"To reply in the current conversation, respond with plain text content — do NOT call the message tool. " +
				"The message tool is ONLY for sending notifications to OTHER channels (e.g., user is in desktop but you want to ping their Telegram). " +
				"Try again: either respond as plain text, or specify a different channel.",
			IsError: true,
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
