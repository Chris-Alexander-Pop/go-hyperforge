package pinecone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Store implements vector.Store for Pinecone.
type Store struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// New creates a new Pinecone vector store.
func New(cfg vector.Config) (*Store, error) {
	// Construct Base URL based on inputs or use explicit Host if provided
	baseURL := cfg.Host
	if baseURL == "" {
		// Fallback to standard Pinecone URL structure
		// https://index-project.svc.environment.pinecone.io
		baseURL = fmt.Sprintf("https://%s-%s.svc.%s.pinecone.io",
			cfg.IndexName, cfg.ProjectID, cfg.Environment)
	}

	return &Store{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Search finds the nearest neighbors to the query vector.
func (s *Store) Search(ctx context.Context, queryVector []float32, limit int) ([]vector.Result, error) {
	return s.SearchWithOpts(ctx, queryVector, vector.SearchOpts{Limit: limit})
}

// SearchWithOpts finds nearest neighbors with optional metadata filter (Pinecone filter).
func (s *Store) SearchWithOpts(ctx context.Context, queryVector []float32, opts vector.SearchOpts) ([]vector.Result, error) {
	url := fmt.Sprintf("%s/query", s.baseURL)

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	reqBody := map[string]interface{}{
		"vector":          queryVector,
		"topK":            limit,
		"includeMetadata": true,
	}
	if len(opts.Filter) > 0 {
		reqBody["filter"] = opts.Filter
	}

	resp, err := s.doRequest(ctx, "POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleError(resp)
	}

	var result struct {
		Matches []struct {
			ID       string                 `json:"id"`
			Score    float32                `json:"score"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"matches"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode pinecone response")
	}

	vecResults := make([]vector.Result, len(result.Matches))
	for i, m := range result.Matches {
		vecResults[i] = vector.Result{
			ID:       m.ID,
			Score:    m.Score,
			Metadata: m.Metadata,
		}
	}

	return vecResults, nil
}

// Upsert inserts or updates a vector with metadata.
func (s *Store) Upsert(ctx context.Context, id string, vec []float32, metadata map[string]interface{}) error {
	url := fmt.Sprintf("%s/vectors/upsert", s.baseURL)

	reqBody := map[string]interface{}{
		"vectors": []map[string]interface{}{
			{
				"id":       id,
				"values":   vec,
				"metadata": metadata,
			},
		},
	}

	resp, err := s.doRequest(ctx, "POST", url, reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.handleError(resp)
	}
	return nil
}

// Delete removes a vector by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/vectors/delete", s.baseURL)

	reqBody := map[string]interface{}{
		"ids": []string{id},
	}

	resp, err := s.doRequest(ctx, "POST", url, reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.handleError(resp)
	}
	return nil
}

// Close is a no-op for HTTP client.
func (s *Store) Close() error {
	return nil
}

func (s *Store) doRequest(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal request body")
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", s.apiKey)

	return s.client.Do(req)
}

func (s *Store) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return errors.New(errors.CodeInternal, fmt.Sprintf("pinecone api error: %s", string(body)), nil)
}

// Ensure Store implements vector.Store
var _ vector.Store = (*Store)(nil)
