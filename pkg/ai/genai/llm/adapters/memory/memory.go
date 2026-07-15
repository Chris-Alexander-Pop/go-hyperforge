// Package memory provides an in-memory LLM client for testing.
package memory

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
)

// Client is an in-memory LLM client for testing purposes.
// It returns predictable responses based on input patterns.
type Client struct {
	responses  map[string]string
	counter    int
	chunkRunes int // StreamChat chunk size in runes; default 8
}

// New creates a new in-memory LLM client.
func New() *Client {
	return &Client{
		responses:  make(map[string]string),
		chunkRunes: 8,
	}
}

// WithResponse adds a canned response for a given input pattern.
func (c *Client) WithResponse(pattern, response string) *Client {
	c.responses[pattern] = response
	return c
}

// WithChunkSize sets StreamChat chunk size in Unicode runes (minimum 1).
func (c *Client) WithChunkSize(n int) *Client {
	if n < 1 {
		n = 1
	}
	c.chunkRunes = n
	return c
}

// Chat implements the llm.Client interface.
func (c *Client) Chat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (*llm.Generation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, llm.ErrEmptyMessages
	}

	content := c.resolveContent(messages)
	return c.generateResponse(content), nil
}

// StreamChat streams the assistant response in rune-sized chunks for tests.
func (c *Client) StreamChat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (<-chan llm.GenerationChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, llm.ErrEmptyMessages
	}

	content := c.resolveContent(messages)
	gen := c.generateResponse(content)
	size := c.chunkRunes
	if size < 1 {
		size = 8
	}

	ch := make(chan llm.GenerationChunk)
	go func() {
		defer close(ch)
		runes := []rune(gen.Message.Content)
		for i := 0; i < len(runes); i += size {
			if err := ctx.Err(); err != nil {
				select {
				case ch <- llm.GenerationChunk{Err: err}:
				default:
				}
				return
			}
			end := i + size
			if end > len(runes) {
				end = len(runes)
			}
			delta := string(runes[i:end])
			chunk := llm.GenerationChunk{Delta: delta}
			if end == len(runes) {
				chunk.FinishReason = gen.FinishReason
				usage := gen.Usage
				chunk.Usage = &usage
			}
			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}
		}
		// Empty content: still emit a terminal chunk.
		if len(runes) == 0 {
			usage := gen.Usage
			select {
			case ch <- llm.GenerationChunk{FinishReason: gen.FinishReason, Usage: &usage}:
			case <-ctx.Done():
			}
		}
	}()
	return ch, nil
}

func (c *Client) resolveContent(messages []llm.Message) string {
	var lastContent string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == llm.RoleUser {
			lastContent = messages[i].Content
			break
		}
	}

	for pattern, response := range c.responses {
		if strings.Contains(strings.ToLower(lastContent), strings.ToLower(pattern)) {
			return response
		}
	}

	c.counter++
	return fmt.Sprintf("Memory LLM response #%d: Echo of '%s'", c.counter, truncate(lastContent, 50))
}

func (c *Client) generateResponse(content string) *llm.Generation {
	promptTokens := 10 // Simulated
	completionTokens := len(strings.Fields(content))
	if completionTokens == 0 && content != "" {
		completionTokens = utf8.RuneCountInString(content)
	}

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
