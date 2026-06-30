package groq

import (
	"sync"
)

// TokenUsageEvent represents the details of a Groq API token usage
type TokenUsageEvent struct {
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
}

// Broadcaster manages active streaming connections (subscribers)
type Broadcaster struct {
	mu          sync.Mutex
	subscribers map[chan TokenUsageEvent]bool
}

// GlobalBroadcaster is the package-level instance used by the client and API handler
var GlobalBroadcaster = NewBroadcaster()

// NewBroadcaster creates a new Broadcaster instance
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan TokenUsageEvent]bool),
	}
}

// Subscribe adds a new client channel to the subscriber list
func (b *Broadcaster) Subscribe() chan TokenUsageEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan TokenUsageEvent, 10)
	b.subscribers[ch] = true
	return ch
}

// Unsubscribe removes a client channel and closes it
func (b *Broadcaster) Unsubscribe(ch chan TokenUsageEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.subscribers[ch]; exists {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Publish broadcasts a TokenUsageEvent to all active subscribers
func (b *Broadcaster) Publish(event TokenUsageEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscribers {
		// Non-blocking send to avoid locking the client if a subscriber is slow
		select {
		case ch <- event:
		default:
		}
	}
}
