// Package memory provides context management for LLM conversations.
package memory

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Memory manages conversation history.
// All methods accept context for cancellation and future store-backed implementations.
type Memory interface {
	// AddMessage adds a message to the history (text or multimodal).
	AddMessage(ctx context.Context, message llm.Message) error

	// AddUserMessage adds a plain-text user message.
	AddUserMessage(ctx context.Context, content string) error

	// AddAssistantMessage adds an assistant message.
	AddAssistantMessage(ctx context.Context, content string) error

	// AddUserParts adds a multimodal user message from content parts.
	AddUserParts(ctx context.Context, parts ...llm.ContentPart) error

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
	if msg.Content == "" && len(msg.Parts) == 0 && len(msg.ToolCalls) == 0 {
		return errors.InvalidArgument("message content or parts required", nil)
	}
	// Normalize: if only Parts are set, mirror text into Content for text-only consumers.
	if msg.Content == "" && len(msg.Parts) > 0 {
		msg.Content = msg.TextContent()
	}
	m.messages = append(m.messages, cloneMessage(msg))
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

func (m *SimpleMemory) AddUserParts(ctx context.Context, parts ...llm.ContentPart) error {
	if len(parts) == 0 {
		return errors.InvalidArgument("at least one content part is required", nil)
	}
	copied := make([]llm.ContentPart, len(parts))
	copy(copied, parts)
	return m.AddMessage(ctx, llm.Message{Role: llm.RoleUser, Parts: copied})
}

func (m *SimpleMemory) GetMessages(ctx context.Context) ([]llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	msgs := make([]llm.Message, len(m.messages))
	for i, msg := range m.messages {
		msgs[i] = cloneMessage(msg)
	}
	return msgs, nil
}

func (m *SimpleMemory) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.messages = make([]llm.Message, 0)
	return nil
}

func cloneMessage(msg llm.Message) llm.Message {
	out := msg
	if len(msg.Parts) > 0 {
		out.Parts = make([]llm.ContentPart, len(msg.Parts))
		copy(out.Parts, msg.Parts)
		for i, p := range msg.Parts {
			if len(p.Data) > 0 {
				out.Parts[i].Data = append([]byte(nil), p.Data...)
			}
		}
	}
	if msg.Metadata != nil {
		out.Metadata = make(map[string]interface{}, len(msg.Metadata))
		for k, v := range msg.Metadata {
			out.Metadata[k] = v
		}
	}
	if len(msg.ToolCalls) > 0 {
		out.ToolCalls = append([]llm.ToolCall(nil), msg.ToolCalls...)
	}
	return out
}

var _ Memory = (*SimpleMemory)(nil)
