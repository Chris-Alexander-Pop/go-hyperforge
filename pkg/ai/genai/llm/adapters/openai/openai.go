// Package openai provides an OpenAI Adapter.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
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

	// Map LLM messages to OpenAI format
	// OpenAI separates ToolCalls from Content slightly differently in API,
	// but mostly it maps directly now.

	reqBody := map[string]interface{}{
		"model":       options.Model,
		"messages":    messages,
		"temperature": options.Temperature,
	}
	if len(options.Tools) > 0 {
		reqBody["tools"] = options.Tools
	}

	jsonBody, _ := json.Marshal(reqBody)
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

var _ llm.Client = (*Client)(nil)
