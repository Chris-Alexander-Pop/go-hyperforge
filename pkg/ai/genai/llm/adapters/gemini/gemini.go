// Package gemini provides a Google Gemini Adapter.
package gemini

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Client struct {
	client *genai.Client
}

func New(apiKey string) (*Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, pkgerrors.Internal("failed to create gemini client", err)
	}
	return &Client{client: client}, nil
}

func (c *Client) Chat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (*llm.Generation, error) {
	options := llm.GenerateOptions{
		Model:       "gemini-1.5-pro",
		Temperature: 0.7,
	}
	for _, o := range opts {
		o(&options)
	}

	model := c.client.GenerativeModel(options.Model)
	model.SetTemperature(float32(options.Temperature))
	if options.MaxTokens > 0 {
		model.SetMaxOutputTokens(int32(options.MaxTokens))
	}

	// Convert history
	cs := model.StartChat()

	// Gemini ChatSession history is []*genai.Content
	// But we have to set it manually or send history one by one?
	// The SDK manages history in StartChat if we use SendMessage, but we are stateless here mostly.
	// So we need to reconstruct history.

	var history []*genai.Content
	// Last message is the new prompt
	if len(messages) == 0 {
		return nil, pkgerrors.InvalidArgument("empty messages", nil)
	}

	// Separate history (0 to N-1) and current prompt (N)
	// Gemini separates System instructions now

	for i, m := range messages {
		// If last message, don't add to history, it's the prompt
		if i == len(messages)-1 && m.Role == llm.RoleUser {
			continue // handled as SendMessage argument
		}

		if m.Role == llm.RoleSystem {
			model.SystemInstruction = &genai.Content{
				Parts: []genai.Part{genai.Text(m.Content)},
			}
			continue
		}

		role := "user"
		if m.Role == llm.RoleAssistant {
			role = "model"
		}

		history = append(history, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(m.Content)},
		})
	}

	cs.History = history

	lastMsg := messages[len(messages)-1]
	resp, err := cs.SendMessage(ctx, genai.Text(lastMsg.Content))
	if err != nil {
		return nil, pkgerrors.Internal("gemini request failed", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, pkgerrors.Internal("no candidates returned", nil)
	}

	// Extract text
	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if t, ok := part.(genai.Text); ok {
			text += string(t)
		}
	}

	return &llm.Generation{
		Message: llm.Message{
			Role:    llm.RoleAssistant,
			Content: text,
		},
		FinishReason: string(resp.Candidates[0].FinishReason),
		// Token usage is in resp.UsageMetadata
	}, nil
}

func (c *Client) Close() {
	c.client.Close()
}

var _ llm.Client = (*Client)(nil)
