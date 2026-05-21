package channels

import (
	"context"
	"strings"
	"sync"

	"github.com/Agentx-network/agentx/pkg/bus"
	"github.com/Agentx-network/agentx/pkg/logger"
)

type Channel interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Send(ctx context.Context, msg bus.OutboundMessage) error
	IsRunning() bool
	IsAllowed(senderID string) bool
	SetOwnerClaimHook(func(channelName, senderID string))
}

// StreamableChannel is an optional interface for channels that support streaming deltas.
type StreamableChannel interface {
	Channel
	HandleStreamDelta(delta bus.StreamDelta)
	StartStreamConsumer(ctx context.Context)
}

type BaseChannel struct {
	config     any
	bus        *bus.MessageBus
	running    bool
	name       string
	allowList  []string
	mu         sync.Mutex
	ownerClaim func(channelName, senderID string) // persists the first sender as owner
}

func NewBaseChannel(name string, config any, bus *bus.MessageBus, allowList []string) *BaseChannel {
	// H1 (audit): an enabled channel with no allow-list now rejects every
	// sender (fail closed). Warn loudly so the user knows they must add their
	// own ID (or "*" for open access) before the channel will respond.
	if len(allowList) == 0 {
		logger.WarnCF("channels", "Channel enabled with an empty allow-list — it will reject ALL senders until allow_from is set (use \"*\" to allow everyone)",
			map[string]any{"channel": name})
	}
	return &BaseChannel{
		config:    config,
		bus:       bus,
		name:      name,
		allowList: allowList,
		running:   false,
	}
}

func (c *BaseChannel) Name() string {
	return c.name
}

func (c *BaseChannel) IsRunning() bool {
	return c.running
}

func (c *BaseChannel) IsAllowed(senderID string) bool {
	// H1 (audit): fail closed. An empty allow-list now means "nobody", not
	// "everyone" — a freshly-enabled bot must not talk to anyone who discovers
	// it. To intentionally allow all senders, set allow_from to ["*"].
	if len(c.allowList) == 0 {
		return false
	}

	// Extract parts from compound senderID like "123456|username"
	idPart := senderID
	userPart := ""
	if idx := strings.Index(senderID, "|"); idx > 0 {
		idPart = senderID[:idx]
		userPart = senderID[idx+1:]
	}

	for _, allowed := range c.allowList {
		// Explicit opt-in to open access.
		if strings.TrimSpace(allowed) == "*" {
			return true
		}
		// Strip leading "@" from allowed value for username matching
		trimmed := strings.TrimPrefix(allowed, "@")
		allowedID := trimmed
		allowedUser := ""
		if idx := strings.Index(trimmed, "|"); idx > 0 {
			allowedID = trimmed[:idx]
			allowedUser = trimmed[idx+1:]
		}

		// Support either side using "id|username" compound form.
		// This keeps backward compatibility with legacy Telegram allowlist entries.
		if senderID == allowed ||
			idPart == allowed ||
			senderID == trimmed ||
			idPart == trimmed ||
			idPart == allowedID ||
			(allowedUser != "" && senderID == allowedUser) ||
			(userPart != "" && (userPart == allowed || userPart == trimmed || userPart == allowedUser)) {
			return true
		}
	}

	return false
}

// SetOwnerClaimHook installs a callback used to persist the first sender as the
// channel's owner (see HandleMessage). Set by the channel manager.
func (c *BaseChannel) SetOwnerClaimHook(fn func(channelName, senderID string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ownerClaim = fn
}

func (c *BaseChannel) HandleMessage(senderID, chatID, content string, media []string, metadata map[string]string) {
	c.mu.Lock()
	emptyAllow := len(c.allowList) == 0
	claim := c.ownerClaim
	c.mu.Unlock()

	if emptyAllow {
		// First-message owner claim: the channel has no allow-list yet, so the
		// FIRST person to message it becomes the owner. We capture their ID
		// (so we can both restrict access to them and proactively notify them
		// later — e.g. reminders), allow this message through, and from now on
		// the channel is locked to that owner (fail-closed for everyone else).
		c.mu.Lock()
		c.allowList = append(c.allowList, senderID)
		c.mu.Unlock()
		logger.InfoCF("channels", "Owner claimed by first message — channel now locked to this sender",
			map[string]any{"channel": c.name, "sender": senderID})
		if claim != nil {
			claim(c.name, senderID)
		}
	} else if !c.IsAllowed(senderID) {
		return
	}

	msg := bus.InboundMessage{
		Channel:  c.name,
		SenderID: senderID,
		ChatID:   chatID,
		Content:  content,
		Media:    media,
		Metadata: metadata,
	}

	c.bus.PublishInbound(msg)
}

func (c *BaseChannel) setRunning(running bool) {
	c.running = running
}
