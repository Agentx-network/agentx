package channels

import (
	"sync"
	"time"
)

// chatRateLimiter is a per-(channel, chatID) sliding-window throttle for
// outbound messages. It's a defense-in-depth measure: if a tool runs away (a
// stuck cron, a subagent loop, anything that publishes too fast), the user's
// chat doesn't drown in messages while we figure it out. Normal use never hits
// the cap — a chatty user sending a few reminders + responses + firings stays
// well below 20/min per chat. When the cap is exceeded, excess messages are
// dropped silently (with a log line) so the chat stays usable.
type chatRateLimiter struct {
	buckets      sync.Map // key "channel:chatID" -> *chatRateBucket
	maxPerMinute int
}

type chatRateBucket struct {
	mu       sync.Mutex
	count    int
	windowAt time.Time
}

func newChatRateLimiter(maxPerMinute int) *chatRateLimiter {
	return &chatRateLimiter{maxPerMinute: maxPerMinute}
}

// allow returns true if a message to this chat is within the per-minute cap.
// false means "drop this message; the chat is being spammed". An empty chatID
// is always allowed (system/internal messages that don't have a chat scope).
func (rl *chatRateLimiter) allow(channel, chatID string) bool {
	if chatID == "" {
		return true
	}
	key := channel + ":" + chatID
	v, _ := rl.buckets.LoadOrStore(key, &chatRateBucket{windowAt: time.Now()})
	b := v.(*chatRateBucket)
	b.mu.Lock()
	defer b.mu.Unlock()

	if time.Since(b.windowAt) >= time.Minute {
		b.windowAt = time.Now()
		b.count = 0
	}
	if b.count >= rl.maxPerMinute {
		return false
	}
	b.count++
	return true
}
