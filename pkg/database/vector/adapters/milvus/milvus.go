package milvus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Store implements vector.Store for Milvus via REST.
type Store struct {
	baseURL        string
	apiKey         string
	collectionName string
	client         *http.Client
}

// New creates a Milvus vector store from vector.Config.
// Host should be the Milvus REST base URL (e.g. http://localhost:19530).
// IndexName maps to the collection name.
func New(cfg vector.Config) (*Store, error) {
	host := strings.TrimRight(cfg.Host, "/")
	if host == "" {
		return nil, errors.InvalidArgument("milvus host is required", nil)
	}
	coll := cfg.IndexName
	if coll == "" {
		coll = "documents"
	}
	return &Store{
		baseURL:        host,
		apiKey:         cfg.APIKey,
		collectionName: coll,
		client:         &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// WithHTTPClient overrides the HTTP client (tests).
func (s *Store) WithHTTPClient(c *http.Client) *Store {
	if c != nil {
		s.client = c
	}
	return s
}

// Search finds nearest neighbors.
func (s *Store) Search(ctx context.Context, queryVector []float32, limit int) ([]vector.Result, error) {
	return s.SearchWithOpts(ctx, queryVector, vector.SearchOpts{Limit: limit})
}

// SearchWithOpts finds nearest neighbors with optional metadata filter.
func (s *Store) SearchWithOpts(ctx context.Context, queryVector []float32, opts vector.SearchOpts) ([]vector.Result, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	body := map[string]interface{}{
		"collectionName": s.collectionName,
		"data":           [][]float32{queryVector},
		"limit":          limit,
		"outputFields":   []string{"*"},
	}
	if len(opts.Filter) > 0 {
		parts := make([]string, 0, len(opts.Filter))
		for k, v := range opts.Filter {
			parts = append(parts, fmt.Sprintf(`%s == %q`, k, fmt.Sprint(v)))
		}
		body["filter"] = strings.Join(parts, " && ")
	}

	resp, err := s.doRequest(ctx, "POST", s.baseURL+"/v2/vectordb/entities/search", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleError(resp)
	}

	var parsed struct {
		Code int `json:"code"`
		Data []struct {
			ID       interface{}            `json:"id"`
			Distance float32                `json:"distance"`
			Score    float32                `json:"score"`
			Entity   map[string]interface{} `json:"entity"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, errors.Wrap(err, "failed to decode milvus search response")
	}
	if parsed.Code != 0 && parsed.Code != 200 {
		msg := parsed.Message
		if msg == "" {
			msg = "milvus search failed"
		}
		return nil, errors.Internal(msg, nil)
	}

	out := make([]vector.Result, 0, len(parsed.Data))
	for _, row := range parsed.Data {
		res := vector.Result{Metadata: map[string]interface{}{}}
		res.ID = fmt.Sprint(row.ID)
		if row.Score != 0 {
			res.Score = row.Score
		} else {
			// Milvus distance: lower is closer; convert to similarity-ish score.
			res.Score = 1.0 / (1.0 + row.Distance)
		}
		for k, v := range row.Entity {
			if k == "id" || k == "vector" {
				continue
			}
			res.Metadata[k] = v
		}
		out = append(out, res)
	}
	return out, nil
}

// Upsert inserts or updates a vector with metadata.
func (s *Store) Upsert(ctx context.Context, id string, vec []float32, metadata map[string]interface{}) error {
	row := map[string]interface{}{
		"id":     id,
		"vector": vec,
	}
	for k, v := range metadata {
		row[k] = v
	}
	body := map[string]interface{}{
		"collectionName": s.collectionName,
		"data":           []map[string]interface{}{row},
	}

	resp, err := s.doRequest(ctx, "POST", s.baseURL+"/v2/vectordb/entities/upsert", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return s.handleError(resp)
	}

	var parsed struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	if parsed.Code != 0 && parsed.Code != 200 {
		msg := parsed.Message
		if msg == "" {
			msg = "milvus upsert failed"
		}
		return errors.Internal(msg, nil)
	}
	return nil
}

// Delete removes a vector by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	body := map[string]interface{}{
		"collectionName": s.collectionName,
		"filter":         fmt.Sprintf(`id == %q`, id),
	}

	resp, err := s.doRequest(ctx, "POST", s.baseURL+"/v2/vectordb/entities/delete", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return vector.ErrVectorNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return s.handleError(resp)
	}

	var parsed struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	if parsed.Code == 404 {
		return vector.ErrVectorNotFound
	}
	if parsed.Code != 0 && parsed.Code != 200 {
		msg := parsed.Message
		if msg == "" {
			msg = "milvus delete failed"
		}
		return errors.Internal(msg, nil)
	}
	return nil
}

// Close is a no-op for the HTTP client.
func (s *Store) Close() error { return nil }

func (s *Store) doRequest(ctx context.Context, method, urlStr string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal milvus request")
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create milvus request")
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	return s.client.Do(req)
}

func (s *Store) handleError(resp *http.Response) error {
	b, _ := io.ReadAll(resp.Body)
	return errors.Internal(fmt.Sprintf("milvus api error (%d): %s", resp.StatusCode, string(b)), nil)
}

var _ vector.Store = (*Store)(nil)
