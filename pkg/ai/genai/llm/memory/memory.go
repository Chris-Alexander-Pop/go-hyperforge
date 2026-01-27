// Package memory provides context management for LLM conversations.
package memory

import "github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"

// Memory manages conversation history.
type Memory interface {
	// AddMessage adds a message to the history.
	AddMessage(message llm.Message)

	// AddUserMessage adds a user message.
	AddUserMessage(content string)

	// AddAssistantMessage adds an assistant message.
	AddAssistantMessage(content string)

	// GetMessages returns the current history.
	GetMessages() []llm.Message

	// Clear resets the history.
	Clear()
}

// SimpleMemory is an in-memory buffer.
type SimpleMemory struct {
	messages []llm.Message
	maxLen   int
}

func NewSimpleMemory(maxLen int) *SimpleMemory {
	return &SimpleMemory{
		messages: make([]llm.Message, 0),
		maxLen:   maxLen,
	}
}

func (m *SimpleMemory) AddMessage(msg llm.Message) {
	m.messages = append(m.messages, msg)
	if m.maxLen > 0 && len(m.messages) > m.maxLen {
		m.messages = m.messages[len(m.messages)-m.maxLen:]
	}
}

func (m *SimpleMemory) AddUserMessage(content string) {
	m.AddMessage(llm.Message{Role: llm.RoleUser, Content: content})
}

func (m *SimpleMemory) AddAssistantMessage(content string) {
	m.AddMessage(llm.Message{Role: llm.RoleAssistant, Content: content})
}

func (m *SimpleMemory) GetMessages() []llm.Message {
	// Return copy
	msgs := make([]llm.Message, len(m.messages))
	copy(msgs, m.messages)
	return msgs
}

func (m *SimpleMemory) Clear() {
	m.messages = make([]llm.Message, 0)
}
