// Package openai provides an OpenAI embedding adapter.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/nlp/embedding"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Service implements embedding.Service using OpenAI.
type Service struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// New creates a new OpenAI embedding service.
func New(apiKey string, model string) *Service {
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &Service{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"input": texts,
		"model": s.model,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(reqBody))
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
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, pkgerrors.Internal("failed to parse response", err)
	}

	embeddings := make([][]float32, len(texts))
	for _, item := range result.Data {
		if item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	return embeddings, nil
}

func (s *Service) Dimension() int {
	// Approximate dimensions for common models
	if s.model == "text-embedding-3-small" {
		return 1536
	}
	if s.model == "text-embedding-3-large" {
		return 3072
	}
	return 1536
}

var _ embedding.Service = (*Service)(nil)
