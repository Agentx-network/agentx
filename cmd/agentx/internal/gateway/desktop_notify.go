package gateway

import "sync"

// desktopNotifier buffers messages proactively delivered to the desktop chat —
// cron reminders, async task results — which has no persistent push connection.
// The gateway enqueues them when an outbound message targets the "desktop"
// channel; the desktop app drains them by polling /api/notifications.
type desktopNotifier struct {
	mu    sync.Mutex
	queue map[string][]string // chatID -> pending messages (FIFO)
}

func newDesktopNotifier() *desktopNotifier {
	return &desktopNotifier{queue: map[string][]string{}}
}

// enqueue appends a message for the given desktop chat.
func (n *desktopNotifier) enqueue(chatID, content string) {
	if chatID == "" {
		chatID = "chat"
	}
	n.mu.Lock()
	n.queue[chatID] = append(n.queue[chatID], content)
	n.mu.Unlock()
}

// drain returns and clears all pending messages for the given desktop chat.
func (n *desktopNotifier) drain(chatID string) []string {
	if chatID == "" {
		chatID = "chat"
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	msgs := n.queue[chatID]
	delete(n.queue, chatID)
	return msgs
}
