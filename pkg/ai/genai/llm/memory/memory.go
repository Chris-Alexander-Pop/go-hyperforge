// Package memory provides context management for LLM conversations.
package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
)

// Memory manages conversation history.
// All methods accept context for cancellation and future store-backed implementations.
type Memory interface {
	// AddMessage adds a message to the history.
	AddMessage(ctx context.Context, message llm.Message) error

	// AddUserMessage adds a user message.
	AddUserMessage(ctx context.Context, content string) error

	// AddAssistantMessage adds an assistant message.
	AddAssistantMessage(ctx context.Context, content string) error

	// GetMessages returns the current history.
	GetMessages(ctx context.Context) ([]llm.Message, error)

	// Clear resets the history.
	Clear(ctx context.Context) error
}

// SimpleMemory is an in-memory buffer.
type SimpleMemory struct {
	messages []llm.Message
	maxLen   int
}

// NewSimpleMemory creates a conversation buffer truncated to maxLen messages (0 = unlimited).
func NewSimpleMemory(maxLen int) *SimpleMemory {
	return &SimpleMemory{
		messages: make([]llm.Message, 0),
		maxLen:   maxLen,
	}
}

func (m *SimpleMemory) AddMessage(ctx context.Context, msg llm.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.messages = append(m.messages, msg)
	if m.maxLen > 0 && len(m.messages) > m.maxLen {
		m.messages = m.messages[len(m.messages)-m.maxLen:]
	}
	return nil
}

func (m *SimpleMemory) AddUserMessage(ctx context.Context, content string) error {
	return m.AddMessage(ctx, llm.Message{Role: llm.RoleUser, Content: content})
}

func (m *SimpleMemory) AddAssistantMessage(ctx context.Context, content string) error {
	return m.AddMessage(ctx, llm.Message{Role: llm.RoleAssistant, Content: content})
}

func (m *SimpleMemory) GetMessages(ctx context.Context) ([]llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	msgs := make([]llm.Message, len(m.messages))
	copy(msgs, m.messages)
	return msgs, nil
}

func (m *SimpleMemory) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.messages = make([]llm.Message, 0)
	return nil
}

var _ Memory = (*SimpleMemory)(nil)
