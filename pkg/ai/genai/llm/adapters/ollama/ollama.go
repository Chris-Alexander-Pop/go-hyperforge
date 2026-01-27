// Package ollama provides a local Ollama Adapter.
package ollama

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
	host       string
	httpClient *http.Client
}

func New(host string) *Client {
	if host == "" {
		host = "http://localhost:11434"
	}
	return &Client{
		host:       host,
		httpClient: &http.Client{Timeout: 300 * time.Second}, // Local LLMs can be slow
	}
}

func (c *Client) Chat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (*llm.Generation, error) {
	options := llm.GenerateOptions{
		Model: "llama3",
	}
	for _, o := range opts {
		o(&options)
	}

	// Convert messages
	var ollamaMsgs []map[string]string
	for _, m := range messages {
		ollamaMsgs = append(ollamaMsgs, map[string]string{
			"role":    string(m.Role),
			"content": m.Content,
		})
	}

	reqBody := map[string]interface{}{
		"model":    options.Model,
		"messages": ollamaMsgs,
		"stream":   false,
		"options": map[string]interface{}{
			"temperature": options.Temperature,
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.host+"/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, pkgerrors.Internal("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("ollama connection failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, pkgerrors.Internal("ollama error", nil)
	}

	var result struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done            bool `json:"done"`
		EvalCount       int  `json:"eval_count"`
		PromptEvalCount int  `json:"prompt_eval_count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, pkgerrors.Internal("failed to parse response", err)
	}

	return &llm.Generation{
		Message: llm.Message{
			Role:    llm.Role(result.Message.Role),
			Content: result.Message.Content,
		},
		FinishReason: "stop",
		Usage: llm.Usage{
			PromptTokens:     result.PromptEvalCount,
			CompletionTokens: result.EvalCount,
			TotalTokens:      result.PromptEvalCount + result.EvalCount,
		},
	}, nil
}

var _ llm.Client = (*Client)(nil)
