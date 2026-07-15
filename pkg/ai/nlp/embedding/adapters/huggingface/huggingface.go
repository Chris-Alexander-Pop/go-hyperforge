// Package huggingface provides an HuggingFace Inference embedding adapter.
package huggingface

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/nlp/embedding"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Service implements embedding.Service using HF Inference API.
type Service struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// New creates a new HF embedding service.
func New(apiKey string, model string) *Service {
	return &Service{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	url := fmt.Sprintf("https://api-inference.huggingface.co/pipeline/feature-extraction/%s", s.model)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"inputs": texts,
		"options": map[string]bool{
			"wait_for_model": true,
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
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

	// FH returns a list of lists (embeddings) or list of list of lists?
	// Usually [batch, dim]
	var embeddings [][]float32
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, pkgerrors.Internal("failed to parse response", err)
	}

	return embeddings, nil
}

func (s *Service) Dimension() int {
	return 768 // Default BERT size, usually
}

var _ embedding.Service = (*Service)(nil)
