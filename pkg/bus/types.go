package bus

type InboundMessage struct {
	Channel    string            `json:"channel"`
	SenderID   string            `json:"sender_id"`
	ChatID     string            `json:"chat_id"`
	Content    string            `json:"content"`
	Media      []string          `json:"media,omitempty"`
	SessionKey string            `json:"session_key"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type OutboundMessage struct {
	Channel string `json:"channel"`
	ChatID  string `json:"chat_id"`
	Content string `json:"content"`
}

// StreamDelta represents a streaming text delta for progressive message updates.
type StreamDelta struct {
	Channel string `json:"channel"`
	ChatID  string `json:"chat_id"`
	Delta   string `json:"delta"`
	Done    bool   `json:"done"`
}

type MessageHandler func(InboundMessage) error
