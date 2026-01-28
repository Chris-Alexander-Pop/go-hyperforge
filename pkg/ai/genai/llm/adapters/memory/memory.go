// Package memory provides an in-memory LLM client for testing.
package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
)

// Client is an in-memory LLM client for testing purposes.
// It returns predictable responses based on input patterns.
type Client struct {
	responses map[string]string
	counter   int
}

// New creates a new in-memory LLM client.
func New() *Client {
	return &Client{
		responses: make(map[string]string),
	}
}

// WithResponse adds a canned response for a given input pattern.
func (c *Client) WithResponse(pattern, response string) *Client {
	c.responses[pattern] = response
	return c
}

// Chat implements the llm.Client interface.
// For testing, it returns a predictable response based on the last message content.
func (c *Client) Chat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (*llm.Generation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Get the last user message
	var lastContent string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == llm.RoleUser {
			lastContent = messages[i].Content
			break
		}
	}

	// Check for canned responses
	for pattern, response := range c.responses {
		if strings.Contains(strings.ToLower(lastContent), strings.ToLower(pattern)) {
			return c.generateResponse(response), nil
		}
	}

	// Default echo response for testing
	c.counter++
	defaultResponse := fmt.Sprintf("Memory LLM response #%d: Echo of '%s'", c.counter, truncate(lastContent, 50))

	return c.generateResponse(defaultResponse), nil
}

func (c *Client) generateResponse(content string) *llm.Generation {
	promptTokens := 10 // Simulated
	completionTokens := len(strings.Fields(content))

	return &llm.Generation{
		Message: llm.Message{
			Role:    llm.RoleAssistant,
			Content: content,
		},
		FinishReason: "stop",
		Usage: llm.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

var _ llm.Client = (*Client)(nil)
