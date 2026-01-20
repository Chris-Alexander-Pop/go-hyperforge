package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/chat"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// Sender is an in-memory implementation of the chat.Sender interface.
type Sender struct {
	sentMessages []*chat.Message
	mu           *concurrency.SmartRWMutex
}

// New creates a new memory chat sender.
func New() *Sender {
	return &Sender{
		sentMessages: make([]*chat.Message, 0),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-chat-sender",
		}),
	}
}

// Send stores the chat message in memory.
func (s *Sender) Send(ctx context.Context, msg *chat.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sentMessages = append(s.sentMessages, msg)
	return nil
}

// SentMessages returns a copy of all sent messages.
func (s *Sender) SentMessages() []*chat.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := make([]*chat.Message, len(s.sentMessages))
	copy(msgs, s.sentMessages)
	return msgs
}

// Clear clears the sent messages history.
func (s *Sender) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sentMessages = make([]*chat.Message, 0)
}

// Close implements the chat.Sender interface.
func (s *Sender) Close() error {
	return nil
}
