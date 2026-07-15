// Package openai provides a DALL-E adapter.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/image"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

type Service struct {
	apiKey     string
	httpClient *http.Client
}

func New(apiKey string) *Service {
	return &Service{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (s *Service) Generate(ctx context.Context, prompt string, opts image.Options) ([]string, error) {
	if opts.Model == "" {
		opts.Model = "dall-e-3"
	}
	if opts.Size == "" {
		opts.Size = "1024x1024"
	}

	body := map[string]interface{}{
		"model":  opts.Model,
		"prompt": prompt,
		"n":      opts.N,
		"size":   opts.Size,
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/images/generations", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, pkgerrors.Internal("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("API request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, pkgerrors.Internal("API error", nil)
	}

	var result struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, pkgerrors.Internal("failed to parse response", err)
	}

	urls := make([]string, len(result.Data))
	for i, d := range result.Data {
		urls[i] = d.URL
	}

	return urls, nil
}

var _ image.Service = (*Service)(nil)
