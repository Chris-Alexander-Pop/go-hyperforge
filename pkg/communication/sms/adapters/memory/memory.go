package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/sms"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// Sender is an in-memory implementation of the sms.Sender interface.
type Sender struct {
	sentMessages []*sms.Message
	mu           *concurrency.SmartRWMutex
}

// New creates a new memory SMS sender.
func New() sms.Sender {
	return &Sender{
		sentMessages: make([]*sms.Message, 0),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-sms-sender",
		}),
	}
}

// Send stores the SMS in memory.
func (s *Sender) Send(ctx context.Context, msg *sms.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sentMessages = append(s.sentMessages, msg)
	return nil
}

// SendBatch stores the batch of SMS messages in memory.
func (s *Sender) SendBatch(ctx context.Context, msgs []*sms.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sentMessages = append(s.sentMessages, msgs...)
	return nil
}

// SentMessages returns a copy of all sent messages.
func (s *Sender) SentMessages() []*sms.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := make([]*sms.Message, len(s.sentMessages))
	copy(msgs, s.sentMessages)
	return msgs
}

// Clear clears the sent messages history.
func (s *Sender) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sentMessages = make([]*sms.Message, 0)
}

// Close implements the sms.Sender interface.
func (s *Sender) Close() error {
	return nil
}
