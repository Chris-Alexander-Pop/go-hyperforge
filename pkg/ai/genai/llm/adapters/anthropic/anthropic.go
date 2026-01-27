// Package anthropic provides an Anthropic Adapter.
package anthropic

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
		Model:     "claude-3-sonnet-20240229",
		MaxTokens: 1024,
	}
	for _, o := range opts {
		o(&options)
	}

	var system string
	var anthropicMsgs []map[string]string

	for _, m := range messages {
		if m.Role == llm.RoleSystem {
			system = m.Content
		} else {
			anthropicMsgs = append(anthropicMsgs, map[string]string{
				"role":    string(m.Role),
				"content": m.Content,
			})
		}
	}

	body := map[string]interface{}{
		"model":       options.Model,
		"messages":    anthropicMsgs,
		"max_tokens":  options.MaxTokens,
		"temperature": options.Temperature,
	}
	if system != "" {
		body["system"] = system
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, pkgerrors.Internal("failed to create request", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("API request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, pkgerrors.Internal("API error", nil)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, pkgerrors.Internal("failed to parse response", err)
	}

	content := ""
	if len(result.Content) > 0 {
		content = result.Content[0].Text
	}

	return &llm.Generation{
		Message: llm.Message{
			Role:    llm.RoleAssistant,
			Content: content,
		},
		FinishReason: result.StopReason,
		Usage: llm.Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}, nil
}

var _ llm.Client = (*Client)(nil)
