package channels

import (
	"testing"
	"time"
)

func TestChatRateLimiter_AllowsUpToCap(t *testing.T) {
	rl := newChatRateLimiter(20)
	for i := 1; i <= 20; i++ {
		if !rl.allow("telegram", "12345") {
			t.Fatalf("call #%d should be allowed (cap is 20)", i)
		}
	}
}

func TestChatRateLimiter_BlocksOverCap(t *testing.T) {
	rl := newChatRateLimiter(5)
	for i := 1; i <= 5; i++ {
		if !rl.allow("telegram", "x") {
			t.Fatalf("call #%d should be allowed", i)
		}
	}
	for i := 6; i <= 10; i++ {
		if rl.allow("telegram", "x") {
			t.Errorf("call #%d should be blocked (cap is 5)", i)
		}
	}
}

func TestChatRateLimiter_IsPerChat(t *testing.T) {
	rl := newChatRateLimiter(3)
	// chat A burns its quota
	for i := 0; i < 3; i++ {
		rl.allow("telegram", "A")
	}
	if rl.allow("telegram", "A") {
		t.Error("chat A should be capped")
	}
	// chat B has its own quota
	for i := 0; i < 3; i++ {
		if !rl.allow("telegram", "B") {
			t.Errorf("chat B should NOT be affected by chat A's quota")
		}
	}
}

func TestChatRateLimiter_IsPerChannel(t *testing.T) {
	rl := newChatRateLimiter(2)
	for i := 0; i < 2; i++ {
		rl.allow("telegram", "1")
	}
	if rl.allow("telegram", "1") {
		t.Error("telegram:1 should be capped")
	}
	// Same chat ID on a different channel is a different bucket.
	if !rl.allow("discord", "1") {
		t.Error("discord:1 should NOT inherit telegram:1's bucket")
	}
}

func TestChatRateLimiter_EmptyChatIDIsAlwaysAllowed(t *testing.T) {
	rl := newChatRateLimiter(1)
	// system/internal messages without a chat scope must never be throttled.
	for i := 0; i < 100; i++ {
		if !rl.allow("system", "") {
			t.Fatalf("empty chatID should always be allowed (i=%d)", i)
		}
	}
}

// Verify that the sliding window actually resets after a minute. We can't wait
// a full minute in tests, so we backdate the bucket's windowAt and confirm the
// next allow() resets the counter.
func TestChatRateLimiter_WindowResets(t *testing.T) {
	rl := newChatRateLimiter(2)
	rl.allow("telegram", "x")
	rl.allow("telegram", "x")
	if rl.allow("telegram", "x") {
		t.Fatal("third call should be blocked before window reset")
	}
	// Force the window to look stale.
	v, _ := rl.buckets.Load("telegram:x")
	b := v.(*chatRateBucket)
	b.mu.Lock()
	b.windowAt = time.Now().Add(-2 * time.Minute)
	b.mu.Unlock()

	if !rl.allow("telegram", "x") {
		t.Error("after window reset, calls should be allowed again")
	}
}
