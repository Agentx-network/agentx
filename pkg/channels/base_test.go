package channels

import (
	"context"
	"testing"
	"time"

	"github.com/Agentx-network/agentx/pkg/bus"
)

// First-message owner-claim: a channel with no allow-list locks to the first
// sender (captured via the hook) and rejects everyone else afterward.
func TestOwnerClaimOnFirstMessage(t *testing.T) {
	mb := bus.NewMessageBus()
	ch := NewBaseChannel("telegram", nil, mb, nil) // empty allow-list

	var claimed string
	ch.SetOwnerClaimHook(func(channel, senderID string) { claimed = channel + ":" + senderID })

	// First sender → claimed, allowed, published.
	ch.HandleMessage("1046193410", "1046193410", "hi", nil, nil)
	if claimed != "telegram:1046193410" {
		t.Fatalf("owner not claimed via hook, got %q", claimed)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if msg, ok := mb.ConsumeInbound(ctx); !ok || msg.SenderID != "1046193410" {
		t.Fatalf("first message should be published; ok=%v msg=%+v", ok, msg)
	}
	if !ch.IsAllowed("1046193410") {
		t.Error("owner should be allowed after claim")
	}

	// A different sender is now rejected (not published).
	ch.HandleMessage("999999", "999999", "intruder", nil, nil)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel2()
	if _, ok := mb.ConsumeInbound(ctx2); ok {
		t.Error("a non-owner message must NOT be published after the owner is claimed")
	}
}

func TestBaseChannelIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		allowList []string
		senderID  string
		want      bool
	}{
		{
			name:      "empty allowlist denies all (fail closed)",
			allowList: nil,
			senderID:  "anyone",
			want:      false,
		},
		{
			name:      "wildcard allowlist allows all",
			allowList: []string{"*"},
			senderID:  "anyone",
			want:      true,
		},
		{
			name:      "compound sender matches numeric allowlist",
			allowList: []string{"123456"},
			senderID:  "123456|alice",
			want:      true,
		},
		{
			name:      "compound sender matches username allowlist",
			allowList: []string{"@alice"},
			senderID:  "123456|alice",
			want:      true,
		},
		{
			name:      "numeric sender matches legacy compound allowlist",
			allowList: []string{"123456|alice"},
			senderID:  "123456",
			want:      true,
		},
		{
			name:      "non matching sender is denied",
			allowList: []string{"123456"},
			senderID:  "654321|bob",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := NewBaseChannel("test", nil, nil, tt.allowList)
			if got := ch.IsAllowed(tt.senderID); got != tt.want {
				t.Fatalf("IsAllowed(%q) = %v, want %v", tt.senderID, got, tt.want)
			}
		})
	}
}
