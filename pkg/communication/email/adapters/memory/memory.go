package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// Sender is an in-memory implementation of the email.Sender interface.
// useful for testing and local development.
type Sender struct {
	sentMessages []*email.Message
	mu           *concurrency.SmartRWMutex
}

// New creates a new memory email sender.
func New() email.Sender {
	return &Sender{
		sentMessages: make([]*email.Message, 0),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-email-sender",
		}),
	}
}

// Send stores the email in memory.
func (s *Sender) Send(ctx context.Context, msg *email.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sentMessages = append(s.sentMessages, msg)
	return nil
}

// SendBatch stores the batch of emails in memory.
func (s *Sender) SendBatch(ctx context.Context, msgs []*email.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sentMessages = append(s.sentMessages, msgs...)
	return nil
}

// SentMessages returns a copy of all sent messages.
func (s *Sender) SentMessages() []*email.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := make([]*email.Message, len(s.sentMessages))
	copy(msgs, s.sentMessages)
	return msgs
}

// Clear clears the sent messages history.
func (s *Sender) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sentMessages = make([]*email.Message, 0)
}

// Close implements the email.Sender interface.
func (s *Sender) Close() error {
	return nil
}
