package gui

import (
	"sync"
)

// LogBroadcaster manages subscribers for live console and metric event streaming.
type LogBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan string]struct{}
	history     []string
	maxHistory  int
}

// NewLogBroadcaster creates a new LogBroadcaster with circular history buffering.
func NewLogBroadcaster(maxHistory int) *LogBroadcaster {
	if maxHistory <= 0 {
		maxHistory = 100
	}
	return &LogBroadcaster{
		subscribers: make(map[chan string]struct{}),
		history:     make([]string, 0, maxHistory),
		maxHistory:  maxHistory,
	}
}

// Subscribe registers a new subscriber channel and returns recent log history.
func (b *LogBroadcaster) Subscribe() (chan string, []string) {
	ch := make(chan string, 32)

	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[ch] = struct{}{}
	histCopy := make([]string, len(b.history))
	copy(histCopy, b.history)

	return ch, histCopy
}

// Unsubscribe removes a subscriber channel.
func (b *LogBroadcaster) Unsubscribe(ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subscribers, ch)
	close(ch)
}

// Broadcast sends a message to all active subscribers and appends to history.
func (b *LogBroadcaster) Broadcast(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.history) >= b.maxHistory {
		b.history = append(b.history[1:], line)
	} else {
		b.history = append(b.history, line)
	}

	for ch := range b.subscribers {
		select {
		case ch <- line:
		default:
			// Non-blocking drop if consumer is slow
		}
	}
}
