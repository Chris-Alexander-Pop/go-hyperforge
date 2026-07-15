// Package openai provides an OpenAI Adapter.
package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *Client) Chat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (*llm.Generation, error) {
	options := llm.GenerateOptions{
		Model:       "gpt-4-turbo-preview",
		Temperature: 0.7,
	}
	for _, o := range opts {
		o(&options)
	}

	apiMessages := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		apiMessages = append(apiMessages, mapMessage(m))
	}

	reqBody := map[string]interface{}{
		"model":       options.Model,
		"messages":    apiMessages,
		"temperature": options.Temperature,
	}
	if options.MaxTokens > 0 {
		reqBody["max_tokens"] = options.MaxTokens
	}
	if len(options.Tools) > 0 {
		reqBody["tools"] = options.Tools
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, pkgerrors.Internal("failed to marshal request", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, pkgerrors.Internal("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("API request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, pkgerrors.Internal("API error", nil)
	}

	var result struct {
		Choices []struct {
			Message      llm.Message `json:"message"`
			FinishReason string      `json:"finish_reason"`
		} `json:"choices"`
		Usage llm.Usage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, pkgerrors.Internal("failed to parse response", err)
	}

	if len(result.Choices) == 0 {
		return nil, pkgerrors.Internal("no choices", nil)
	}

	return &llm.Generation{
		Message:      result.Choices[0].Message,
		FinishReason: result.Choices[0].FinishReason,
		Usage:        result.Usage,
	}, nil
}

// mapMessage converts an llm.Message into OpenAI chat message JSON,
// including multimodal content arrays when Parts are present.
func mapMessage(m llm.Message) map[string]interface{} {
	out := map[string]interface{}{
		"role": string(m.Role),
	}
	if m.Name != "" {
		out["name"] = m.Name
	}
	if m.ToolCallID != "" {
		out["tool_call_id"] = m.ToolCallID
	}
	if len(m.ToolCalls) > 0 {
		out["tool_calls"] = m.ToolCalls
	}

	if m.IsMultimodal() {
		parts := make([]map[string]interface{}, 0, len(m.Parts))
		for _, p := range m.Parts {
			switch p.Type {
			case llm.PartTypeText:
				parts = append(parts, map[string]interface{}{
					"type": "text",
					"text": p.Text,
				})
			case llm.PartTypeImageURL:
				parts = append(parts, map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": p.ImageURL,
					},
				})
			case llm.PartTypeImageBase64:
				mime := p.MIMEType
				if mime == "" {
					mime = "image/png"
				}
				dataURL := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(p.Data)
				parts = append(parts, map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": dataURL,
					},
				})
			}
		}
		out["content"] = parts
		return out
	}

	out["content"] = m.Content
	return out
}

// StreamChat adapts Chat into a single-chunk stream until native SSE streaming is wired.
func (c *Client) StreamChat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (<-chan llm.GenerationChunk, error) {
	return llm.StreamFromChat(ctx, c.Chat, messages, opts...)
}

var _ llm.Client = (*Client)(nil)
