// Package weaviate implements vector.Store against the Weaviate REST API.
package weaviate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Store implements vector.Store for Weaviate.
type Store struct {
	baseURL   string
	apiKey    string
	className string
	client    *http.Client
}

// New creates a Weaviate vector store from vector.Config.
// Host should be the Weaviate base URL (e.g. http://localhost:8080).
// IndexName maps to the Weaviate class name.
func New(cfg vector.Config) (*Store, error) {
	host := strings.TrimRight(cfg.Host, "/")
	if host == "" {
		return nil, errors.InvalidArgument("weaviate host is required", nil)
	}
	class := cfg.IndexName
	if class == "" {
		class = "Document"
	}
	return &Store{
		baseURL:   host,
		apiKey:    cfg.APIKey,
		className: class,
		client:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Search finds nearest neighbors via nearVector.
func (s *Store) Search(ctx context.Context, queryVector []float32, limit int) ([]vector.Result, error) {
	return s.SearchWithOpts(ctx, queryVector, vector.SearchOpts{Limit: limit})
}

// SearchWithOpts finds nearest neighbors with optional where-filter on metadata.
func (s *Store) SearchWithOpts(ctx context.Context, queryVector []float32, opts vector.SearchOpts) ([]vector.Result, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	whereClause := ""
	if len(opts.Filter) > 0 {
		parts := make([]string, 0, len(opts.Filter))
		for k, v := range opts.Filter {
			parts = append(parts, fmt.Sprintf(`{path:["%s"] operator:Equal valueText:%q}`, k, fmt.Sprint(v)))
		}
		if len(parts) == 1 {
			whereClause = "where: " + parts[0]
		} else {
			whereClause = "where: {operator:And operands:[" + strings.Join(parts, ",") + "]}"
		}
	}

	// GraphQL nearVector query. Properties include metadata fields plus _additional.
	gql := fmt.Sprintf(`{
  Get {
    %s(
      nearVector: {vector: %s}
      limit: %d
      %s
    ) {
      _additional { id certainty distance }
      text
      source
      doc_id
    }
  }
}`, s.className, formatVector(queryVector), limit, whereClause)

	body := map[string]interface{}{"query": gql}
	resp, err := s.doRequest(ctx, "POST", s.baseURL+"/v1/graphql", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleError(resp)
	}

	var parsed struct {
		Data struct {
			Get map[string][]map[string]interface{} `json:"Get"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, errors.Wrap(err, "failed to decode weaviate response")
	}
	if len(parsed.Errors) > 0 {
		return nil, errors.Internal("weaviate graphql error: "+parsed.Errors[0].Message, nil)
	}

	rows := parsed.Data.Get[s.className]
	out := make([]vector.Result, 0, len(rows))
	for _, row := range rows {
		res := vector.Result{Metadata: map[string]interface{}{}}
		if add, ok := row["_additional"].(map[string]interface{}); ok {
			if id, ok := add["id"].(string); ok {
				res.ID = id
			}
			if cert, ok := add["certainty"].(float64); ok {
				res.Score = float32(cert)
			} else if dist, ok := add["distance"].(float64); ok {
				res.Score = float32(1.0 - dist)
			}
		}
		for k, v := range row {
			if k == "_additional" {
				continue
			}
			res.Metadata[k] = v
		}
		out = append(out, res)
	}
	return out, nil
}

// Upsert creates or replaces an object with a vector.
func (s *Store) Upsert(ctx context.Context, id string, vec []float32, metadata map[string]interface{}) error {
	props := map[string]interface{}{}
	for k, v := range metadata {
		props[k] = v
	}
	body := map[string]interface{}{
		"class":      s.className,
		"id":         id,
		"properties": props,
		"vector":     vec,
	}

	path := fmt.Sprintf("%s/v1/objects/%s", s.baseURL, url.PathEscape(id))
	// Try PUT (replace); if 404, POST create.
	resp, err := s.doRequest(ctx, "PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		createPath := s.baseURL + "/v1/objects"
		resp2, err := s.doRequest(ctx, "POST", createPath, body)
		if err != nil {
			return err
		}
		defer resp2.Body.Close()
		if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
			return s.handleError(resp2)
		}
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return s.handleError(resp)
	}
	return nil
}

// Delete removes an object by UUID/id.
func (s *Store) Delete(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s/v1/objects/%s/%s", s.baseURL, url.PathEscape(s.className), url.PathEscape(id))
	resp, err := s.doRequest(ctx, "DELETE", path, nil)
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
	return nil
}

// Close is a no-op for the HTTP client.
func (s *Store) Close() error { return nil }

func (s *Store) doRequest(ctx context.Context, method, urlStr string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal weaviate request")
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create weaviate request")
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	return s.client.Do(req)
}

func (s *Store) handleError(resp *http.Response) error {
	b, _ := io.ReadAll(resp.Body)
	return errors.Internal(fmt.Sprintf("weaviate api error (%d): %s", resp.StatusCode, string(b)), nil)
}

func formatVector(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

var _ vector.Store = (*Store)(nil)
