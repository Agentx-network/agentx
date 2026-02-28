package bus

import (
	"context"
	"sync"
)

// StreamSubscriber receives copies of stream deltas that match its filter.
type StreamSubscriber struct {
	Ch     chan StreamDelta
	Filter func(StreamDelta) bool // nil means accept all
}

type MessageBus struct {
	inbound    chan InboundMessage
	outbound   chan OutboundMessage
	stream     chan StreamDelta
	streamSubs []*StreamSubscriber
	handlers   map[string]MessageHandler
	closed     bool
	mu         sync.RWMutex
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:  make(chan InboundMessage, 100),
		outbound: make(chan OutboundMessage, 100),
		stream:   make(chan StreamDelta, 500),
		handlers: make(map[string]MessageHandler),
	}
}

func (mb *MessageBus) PublishInbound(msg InboundMessage) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	if mb.closed {
		return
	}
	mb.inbound <- msg
}

func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool) {
	select {
	case msg := <-mb.inbound:
		return msg, true
	case <-ctx.Done():
		return InboundMessage{}, false
	}
}

func (mb *MessageBus) PublishOutbound(msg OutboundMessage) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	if mb.closed {
		return
	}
	mb.outbound <- msg
}

func (mb *MessageBus) SubscribeOutbound(ctx context.Context) (OutboundMessage, bool) {
	select {
	case msg := <-mb.outbound:
		return msg, true
	case <-ctx.Done():
		return OutboundMessage{}, false
	}
}

func (mb *MessageBus) PublishStreamDelta(delta StreamDelta) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	if mb.closed {
		return
	}
	select {
	case mb.stream <- delta:
	default:
		// Drop delta if channel is full to avoid blocking
	}
	// Fan-out to additional subscribers
	for _, sub := range mb.streamSubs {
		if sub.Filter == nil || sub.Filter(delta) {
			select {
			case sub.Ch <- delta:
			default:
			}
		}
	}
}

// AddStreamSubscriber registers an additional stream consumer that receives
// copies of deltas matching its filter. Caller must eventually call
// RemoveStreamSubscriber.
func (mb *MessageBus) AddStreamSubscriber(sub *StreamSubscriber) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.streamSubs = append(mb.streamSubs, sub)
}

// RemoveStreamSubscriber unregisters a subscriber and closes its channel.
func (mb *MessageBus) RemoveStreamSubscriber(sub *StreamSubscriber) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	for i, s := range mb.streamSubs {
		if s == sub {
			mb.streamSubs = append(mb.streamSubs[:i], mb.streamSubs[i+1:]...)
			close(sub.Ch)
			return
		}
	}
}

func (mb *MessageBus) SubscribeStream(ctx context.Context) (StreamDelta, bool) {
	select {
	case delta := <-mb.stream:
		return delta, true
	case <-ctx.Done():
		return StreamDelta{}, false
	}
}

func (mb *MessageBus) RegisterHandler(channel string, handler MessageHandler) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.handlers[channel] = handler
}

func (mb *MessageBus) GetHandler(channel string) (MessageHandler, bool) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	handler, ok := mb.handlers[channel]
	return handler, ok
}

func (mb *MessageBus) Close() {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	if mb.closed {
		return
	}
	mb.closed = true
	close(mb.inbound)
	close(mb.outbound)
	close(mb.stream)
}
