package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/push"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// Sender is an in-memory implementation of the push.Sender interface.
type Sender struct {
	sentMessages []*push.Message
	mu           *concurrency.SmartRWMutex
}

// New creates a new memory push sender.
func New() push.Sender {
	return &Sender{
		sentMessages: make([]*push.Message, 0),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-push-sender",
		}),
	}
}

// Send stores the push notification in memory.
func (s *Sender) Send(ctx context.Context, msg *push.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sentMessages = append(s.sentMessages, msg)
	return nil
}

// SendBatch stores the batch of push notifications in memory.
func (s *Sender) SendBatch(ctx context.Context, msgs []*push.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sentMessages = append(s.sentMessages, msgs...)
	return nil
}

// SentMessages returns a copy of all sent messages.
func (s *Sender) SentMessages() []*push.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := make([]*push.Message, len(s.sentMessages))
	copy(msgs, s.sentMessages)
	return msgs
}

// Clear clears the sent messages history.
func (s *Sender) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sentMessages = make([]*push.Message, 0)
}

// Close implements the push.Sender interface.
func (s *Sender) Close() error {
	return nil
}
