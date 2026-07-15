// Package opensearch implements search.SearchEngine against the OpenSearch REST API.
//
// OpenSearch is Elasticsearch-compatible; this adapter uses pure HTTP (no ES SDK)
// so it can target OpenSearch clusters and httptest doubles independently of
// adapters/elasticsearch.
package opensearch

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

	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Ensure compile-time interface compliance.
var _ search.SearchEngine = (*Engine)(nil)

// Engine implements search.SearchEngine using OpenSearch HTTP.
type Engine struct {
	baseURL  string
	username string
	password string
	apiKey   string
	client   *http.Client
}

// New creates an OpenSearch engine from search.Config.
// Prefers OpenSearchURL when set; otherwise ElasticsearchURL.
func New(cfg search.Config) (*Engine, error) {
	base := strings.TrimRight(cfg.OpenSearchURL, "/")
	if base == "" {
		base = strings.TrimRight(cfg.ElasticsearchURL, "/")
	}
	if base == "" {
		return nil, errors.InvalidArgument("opensearch URL is required", nil)
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Engine{
		baseURL:  base,
		username: cfg.ElasticsearchUsername,
		password: cfg.ElasticsearchPassword,
		apiKey:   cfg.ElasticsearchAPIKey,
		client:   &http.Client{Timeout: timeout},
	}, nil
}

// NewWithConfig is an alias for New.
func NewWithConfig(cfg search.Config) (*Engine, error) {
	return New(cfg)
}

// WithHTTPClient overrides the HTTP client (tests).
func (e *Engine) WithHTTPClient(c *http.Client) *Engine {
	if c != nil {
		e.client = c
	}
	return e
}

// CreateIndex creates an index with optional mappings.
func (e *Engine) CreateIndex(ctx context.Context, indexName string, mapping *search.IndexMapping) error {
	body := map[string]interface{}{}
	if mapping != nil {
		settings := map[string]interface{}{}
		if mapping.Settings.NumberOfShards > 0 {
			settings["number_of_shards"] = mapping.Settings.NumberOfShards
		}
		if mapping.Settings.NumberOfReplicas > 0 {
			settings["number_of_replicas"] = mapping.Settings.NumberOfReplicas
		}
		if len(settings) > 0 {
			body["settings"] = settings
		}
		if len(mapping.Fields) > 0 {
			props := make(map[string]interface{}, len(mapping.Fields))
			for name, fm := range mapping.Fields {
				m := map[string]interface{}{"type": string(fm.Type)}
				if fm.Analyzer != "" {
					m["analyzer"] = fm.Analyzer
				}
				if !fm.Searchable && fm.Type == search.FieldTypeText {
					m["index"] = false
				}
				props[name] = m
			}
			body["mappings"] = map[string]interface{}{"properties": props}
		}
	}
	resp, err := e.do(ctx, http.MethodPut, "/"+url.PathEscape(indexName), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(b), "resource_already_exists") {
			return errors.Conflict("index already exists", nil)
		}
		return errors.InvalidArgument(string(b), nil)
	}
	if resp.StatusCode >= 300 {
		return e.httpErr(resp)
	}
	return nil
}

// DeleteIndex deletes an index.
func (e *Engine) DeleteIndex(ctx context.Context, indexName string) error {
	resp, err := e.do(ctx, http.MethodDelete, "/"+url.PathEscape(indexName), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return errors.NotFound("index not found", nil)
	}
	if resp.StatusCode >= 300 {
		return e.httpErr(resp)
	}
	return nil
}

// GetIndex returns index document count via _stats.
func (e *Engine) GetIndex(ctx context.Context, indexName string) (*search.IndexInfo, error) {
	resp, err := e.do(ctx, http.MethodGet, "/"+url.PathEscape(indexName)+"/_stats", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.NotFound("index not found", nil)
	}
	if resp.StatusCode >= 300 {
		return nil, e.httpErr(resp)
	}
	var result struct {
		Indices map[string]struct {
			Primaries struct {
				Docs struct {
					Count int64 `json:"count"`
				} `json:"docs"`
				Store struct {
					SizeInBytes int64 `json:"size_in_bytes"`
				} `json:"store"`
			} `json:"primaries"`
		} `json:"indices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Internal("failed to decode opensearch stats", err)
	}
	stats, ok := result.Indices[indexName]
	if !ok {
		return nil, errors.NotFound("index not found", nil)
	}
	return &search.IndexInfo{
		Name:      indexName,
		DocCount:  stats.Primaries.Docs.Count,
		SizeBytes: stats.Primaries.Store.SizeInBytes,
	}, nil
}

// Index upserts a document.
func (e *Engine) Index(ctx context.Context, indexName, docID string, doc interface{}) error {
	path := fmt.Sprintf("/%s/_doc/%s", url.PathEscape(indexName), url.PathEscape(docID))
	resp, err := e.do(ctx, http.MethodPut, path, doc)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return e.httpErr(resp)
	}
	return nil
}

// Get retrieves a document by ID.
func (e *Engine) Get(ctx context.Context, indexName, docID string) (*search.Hit, error) {
	path := fmt.Sprintf("/%s/_doc/%s", url.PathEscape(indexName), url.PathEscape(docID))
	resp, err := e.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.NotFound("document not found", nil)
	}
	if resp.StatusCode >= 300 {
		return nil, e.httpErr(resp)
	}
	var result struct {
		ID     string                 `json:"_id"`
		Source map[string]interface{} `json:"_source"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Internal("failed to decode document", err)
	}
	return &search.Hit{ID: result.ID, Score: 1.0, Source: result.Source}, nil
}

// Delete removes a document.
func (e *Engine) Delete(ctx context.Context, indexName, docID string) error {
	path := fmt.Sprintf("/%s/_doc/%s", url.PathEscape(indexName), url.PathEscape(docID))
	resp, err := e.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return errors.NotFound("document not found", nil)
	}
	if resp.StatusCode >= 300 {
		return e.httpErr(resp)
	}
	return nil
}

// Search runs an OpenSearch query.
func (e *Engine) Search(ctx context.Context, indexName string, query search.Query) (*search.SearchResult, error) {
	body := buildSearchQuery(query)
	path := "/" + url.PathEscape(indexName) + "/_search"
	resp, err := e.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.NotFound("index not found", nil)
	}
	if resp.StatusCode >= 300 {
		return nil, e.httpErr(resp)
	}
	return parseSearchResponse(resp.Body)
}

func buildSearchQuery(q search.Query) map[string]interface{} {
	body := map[string]interface{}{
		"from": q.From,
		"size": q.Size,
	}
	if q.Size <= 0 {
		body["size"] = 10
	}
	var must []interface{}
	filter := make([]interface{}, 0, len(q.Filters))
	if q.Text != "" {
		if len(q.Fields) > 0 {
			must = append(must, map[string]interface{}{
				"multi_match": map[string]interface{}{"query": q.Text, "fields": q.Fields},
			})
		} else {
			must = append(must, map[string]interface{}{
				"query_string": map[string]interface{}{"query": q.Text},
			})
		}
	}
	for _, f := range q.Filters {
		filter = append(filter, buildFilter(f))
	}
	if len(must) > 0 || len(filter) > 0 {
		bq := map[string]interface{}{}
		if len(must) > 0 {
			bq["must"] = must
		}
		if len(filter) > 0 {
			bq["filter"] = filter
		}
		body["query"] = map[string]interface{}{"bool": bq}
	} else {
		body["query"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}
	if len(q.Sort) > 0 {
		sorts := make([]interface{}, 0, len(q.Sort))
		for _, s := range q.Sort {
			order := "asc"
			if s.Descending {
				order = "desc"
			}
			sorts = append(sorts, map[string]interface{}{s.Field: map[string]interface{}{"order": order}})
		}
		body["sort"] = sorts
	}
	if q.Highlight {
		body["highlight"] = map[string]interface{}{"fields": map[string]interface{}{"*": map[string]interface{}{}}}
	}
	if len(q.Facets) > 0 {
		aggs := make(map[string]interface{}, len(q.Facets))
		for _, f := range q.Facets {
			aggs[f] = map[string]interface{}{"terms": map[string]interface{}{"field": f}}
		}
		body["aggs"] = aggs
	}
	if q.MinScore > 0 {
		body["min_score"] = q.MinScore
	}
	return body
}

func buildFilter(f search.Filter) map[string]interface{} {
	switch f.Operator {
	case search.FilterOperatorNotEquals:
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{"term": map[string]interface{}{f.Field: f.Value}},
			},
		}
	case search.FilterOperatorGreaterThan:
		return map[string]interface{}{"range": map[string]interface{}{f.Field: map[string]interface{}{"gt": f.Value}}}
	case search.FilterOperatorLessThan:
		return map[string]interface{}{"range": map[string]interface{}{f.Field: map[string]interface{}{"lt": f.Value}}}
	case search.FilterOperatorGreaterOrEq:
		return map[string]interface{}{"range": map[string]interface{}{f.Field: map[string]interface{}{"gte": f.Value}}}
	case search.FilterOperatorLessOrEq:
		return map[string]interface{}{"range": map[string]interface{}{f.Field: map[string]interface{}{"lte": f.Value}}}
	case search.FilterOperatorIn:
		return map[string]interface{}{"terms": map[string]interface{}{f.Field: f.Value}}
	case search.FilterOperatorExists:
		if f.Value == true {
			return map[string]interface{}{"exists": map[string]interface{}{"field": f.Field}}
		}
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{"exists": map[string]interface{}{"field": f.Field}},
			},
		}
	default:
		return map[string]interface{}{"term": map[string]interface{}{f.Field: f.Value}}
	}
}

func parseSearchResponse(body io.Reader) (*search.SearchResult, error) {
	var esResult struct {
		Took int64 `json:"took"`
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			MaxScore float64 `json:"max_score"`
			Hits     []struct {
				ID        string                 `json:"_id"`
				Score     float64                `json:"_score"`
				Source    map[string]interface{} `json:"_source"`
				Highlight map[string][]string    `json:"highlight,omitempty"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations map[string]struct {
			Buckets []struct {
				Key      interface{} `json:"key"`
				DocCount int64       `json:"doc_count"`
			} `json:"buckets"`
		} `json:"aggregations,omitempty"`
	}
	if err := json.NewDecoder(body).Decode(&esResult); err != nil {
		return nil, errors.Internal("failed to decode search response", err)
	}
	result := &search.SearchResult{
		Total:    esResult.Hits.Total.Value,
		MaxScore: esResult.Hits.MaxScore,
		Took:     time.Duration(esResult.Took) * time.Millisecond,
		Hits:     make([]search.Hit, 0, len(esResult.Hits.Hits)),
	}
	for _, hit := range esResult.Hits.Hits {
		result.Hits = append(result.Hits, search.Hit{
			ID: hit.ID, Score: hit.Score, Source: hit.Source, Highlights: hit.Highlight,
		})
	}
	if len(esResult.Aggregations) > 0 {
		result.Facets = make(map[string][]search.FacetValue)
		for name, agg := range esResult.Aggregations {
			vals := make([]search.FacetValue, 0, len(agg.Buckets))
			for _, b := range agg.Buckets {
				vals = append(vals, search.FacetValue{Value: b.Key, Count: b.DocCount})
			}
			result.Facets[name] = vals
		}
	}
	return result, nil
}

// Suggest is not implemented for OpenSearch; use the memory adapter for autocomplete tests.
func (e *Engine) Suggest(ctx context.Context, indexName string, query search.SuggestQuery) ([]search.Suggestion, error) {
	return nil, search.ErrSuggestUnsupported
}

// Bulk performs NDJSON bulk operations.
func (e *Engine) Bulk(ctx context.Context, indexName string, ops []search.BulkOperation) (*search.BulkResult, error) {
	if len(ops) == 0 {
		return &search.BulkResult{}, nil
	}
	var buf bytes.Buffer
	for _, op := range ops {
		meta := map[string]interface{}{"_index": indexName, "_id": op.ID}
		switch op.Action {
		case search.BulkActionDelete:
			_ = json.NewEncoder(&buf).Encode(map[string]interface{}{"delete": meta})
		case search.BulkActionUpdate:
			_ = json.NewEncoder(&buf).Encode(map[string]interface{}{"update": meta})
			_ = json.NewEncoder(&buf).Encode(map[string]interface{}{"doc": op.Document})
		case search.BulkActionCreate:
			_ = json.NewEncoder(&buf).Encode(map[string]interface{}{"create": meta})
			_ = json.NewEncoder(&buf).Encode(op.Document)
		default:
			_ = json.NewEncoder(&buf).Encode(map[string]interface{}{"index": meta})
			_ = json.NewEncoder(&buf).Encode(op.Document)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/_bulk", &buf)
	if err != nil {
		return nil, errors.Internal("failed to create bulk request", err)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	e.applyAuth(req)
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, errors.Internal("opensearch bulk failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, e.httpErr(resp)
	}
	var bulkResp struct {
		Took  int64 `json:"took"`
		Items []map[string]struct {
			ID     string `json:"_id"`
			Status int    `json:"status"`
			Error  *struct {
				Reason string `json:"reason"`
			} `json:"error,omitempty"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		return nil, errors.Internal("failed to decode bulk response", err)
	}
	result := &search.BulkResult{Took: time.Duration(bulkResp.Took) * time.Millisecond}
	for _, item := range bulkResp.Items {
		for _, action := range item {
			if action.Error != nil {
				result.Failed++
				result.Errors = append(result.Errors, search.BulkError{ID: action.ID, Reason: action.Error.Reason})
			} else {
				result.Successful++
			}
		}
	}
	return result, nil
}

// Refresh forces an index refresh.
func (e *Engine) Refresh(ctx context.Context, indexName string) error {
	resp, err := e.do(ctx, http.MethodPost, "/"+url.PathEscape(indexName)+"/_refresh", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return e.httpErr(resp)
	}
	return nil
}

// Close releases resources.
func (e *Engine) Close() error { return nil }

func (e *Engine) do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Internal("failed to encode opensearch body", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, e.baseURL+path, rdr)
	if err != nil {
		return nil, errors.Internal("failed to create opensearch request", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	e.applyAuth(req)
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, errors.Internal("opensearch request failed", err)
	}
	return resp, nil
}

func (e *Engine) applyAuth(req *http.Request) {
	if e.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+e.apiKey)
	} else if e.username != "" {
		req.SetBasicAuth(e.username, e.password)
	}
}

func (e *Engine) httpErr(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	switch resp.StatusCode {
	case http.StatusNotFound:
		return errors.NotFound(msg, nil)
	case http.StatusConflict:
		return errors.Conflict(msg, nil)
	case http.StatusBadRequest:
		return errors.InvalidArgument(msg, nil)
	default:
		return errors.Internal(fmt.Sprintf("opensearch error: %s", msg), nil)
	}
}
