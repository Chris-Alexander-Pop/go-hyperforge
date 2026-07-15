// Package typesense implements search.SearchEngine against the Typesense HTTP API.
package typesense

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Ensure compile-time interface compliance.
var _ search.SearchEngine = (*Engine)(nil)

// Engine implements search.SearchEngine using Typesense REST.
type Engine struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// New creates a Typesense engine from search.Config.
func New(cfg search.Config) (*Engine, error) {
	base := strings.TrimRight(cfg.TypesenseURL, "/")
	if base == "" {
		return nil, errors.InvalidArgument("typesense URL is required", nil)
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Engine{
		baseURL: base,
		apiKey:  cfg.TypesenseAPIKey,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

// NewWithConfig is an alias for New for factory symmetry with other adapters.
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

// CreateIndex creates a Typesense collection from an optional mapping.
func (e *Engine) CreateIndex(ctx context.Context, indexName string, mapping *search.IndexMapping) error {
	fields := []map[string]interface{}{
		{"name": "id", "type": "string"},
	}
	if mapping != nil {
		for name, fm := range mapping.Fields {
			if name == "id" {
				continue
			}
			fields = append(fields, map[string]interface{}{
				"name":     name,
				"type":     typesenseFieldType(fm.Type),
				"facet":    fm.Facetable || fm.Filterable,
				"optional": true,
			})
		}
	}
	body := map[string]interface{}{
		"name":   indexName,
		"fields": fields,
	}
	resp, err := e.do(ctx, http.MethodPost, "/collections", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return errors.Conflict("index already exists", nil)
	}
	if resp.StatusCode >= 300 {
		return e.httpErr(resp)
	}
	return nil
}

func typesenseFieldType(t search.FieldType) string {
	switch t {
	case search.FieldTypeInteger, search.FieldTypeLong:
		return "int64"
	case search.FieldTypeFloat, search.FieldTypeDouble:
		return "float"
	case search.FieldTypeBoolean:
		return "bool"
	case search.FieldTypeGeoPoint:
		return "geopoint"
	default:
		return "string"
	}
}

// DeleteIndex deletes a Typesense collection.
func (e *Engine) DeleteIndex(ctx context.Context, indexName string) error {
	resp, err := e.do(ctx, http.MethodDelete, "/collections/"+url.PathEscape(indexName), nil)
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

// GetIndex returns collection stats.
func (e *Engine) GetIndex(ctx context.Context, indexName string) (*search.IndexInfo, error) {
	resp, err := e.do(ctx, http.MethodGet, "/collections/"+url.PathEscape(indexName), nil)
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
		Name    string `json:"name"`
		NumDocs int64  `json:"num_documents"`
		Created int64  `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Internal("failed to decode typesense collection", err)
	}
	info := &search.IndexInfo{Name: result.Name, DocCount: result.NumDocs}
	if result.Created > 0 {
		info.CreatedAt = time.Unix(result.Created, 0)
	}
	return info, nil
}

// Index upserts a document.
func (e *Engine) Index(ctx context.Context, indexName, docID string, doc interface{}) error {
	docMap, err := toDocMap(doc, docID)
	if err != nil {
		return err
	}
	path := "/collections/" + url.PathEscape(indexName) + "/documents?action=upsert"
	resp, err := e.do(ctx, http.MethodPost, path, docMap)
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

func toDocMap(doc interface{}, docID string) (map[string]interface{}, error) {
	switch d := doc.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(d)+1)
		for k, v := range d {
			out[k] = v
		}
		out["id"] = docID
		return out, nil
	case nil:
		return map[string]interface{}{"id": docID}, nil
	default:
		b, err := json.Marshal(doc)
		if err != nil {
			return nil, errors.Internal("failed to encode document", err)
		}
		var out map[string]interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, errors.Internal("failed to decode document map", err)
		}
		if out == nil {
			out = map[string]interface{}{}
		}
		out["id"] = docID
		return out, nil
	}
}

// Get retrieves a document by ID.
func (e *Engine) Get(ctx context.Context, indexName, docID string) (*search.Hit, error) {
	path := "/collections/" + url.PathEscape(indexName) + "/documents/" + url.PathEscape(docID)
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
	var source map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&source); err != nil {
		return nil, errors.Internal("failed to decode document", err)
	}
	return &search.Hit{ID: docID, Score: 1.0, Source: source}, nil
}

// Delete removes a document.
func (e *Engine) Delete(ctx context.Context, indexName, docID string) error {
	path := "/collections/" + url.PathEscape(indexName) + "/documents/" + url.PathEscape(docID)
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

// Search runs a Typesense search.
func (e *Engine) Search(ctx context.Context, indexName string, query search.Query) (*search.SearchResult, error) {
	q := url.Values{}
	text := query.Text
	if text == "" {
		text = "*"
	}
	q.Set("q", text)
	if len(query.Fields) > 0 {
		q.Set("query_by", strings.Join(query.Fields, ","))
	} else {
		q.Set("query_by", "*")
	}
	size := query.Size
	if size <= 0 {
		size = 10
	}
	q.Set("per_page", strconv.Itoa(size))
	if query.From > 0 && size > 0 {
		q.Set("page", strconv.Itoa(query.From/size+1))
	}
	if len(query.Filters) > 0 {
		parts := make([]string, 0, len(query.Filters))
		for _, f := range query.Filters {
			parts = append(parts, formatFilter(f))
		}
		q.Set("filter_by", strings.Join(parts, " && "))
	}
	if len(query.Sort) > 0 {
		parts := make([]string, 0, len(query.Sort))
		for _, s := range query.Sort {
			dir := "asc"
			if s.Descending {
				dir = "desc"
			}
			parts = append(parts, s.Field+":"+dir)
		}
		q.Set("sort_by", strings.Join(parts, ","))
	}
	if len(query.Facets) > 0 {
		q.Set("facet_by", strings.Join(query.Facets, ","))
	}

	path := "/collections/" + url.PathEscape(indexName) + "/documents/search?" + q.Encode()
	start := time.Now()
	resp, err := e.do(ctx, http.MethodGet, path, nil)
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

	var raw struct {
		Found        int64 `json:"found"`
		SearchTimeMS int   `json:"search_time_ms"`
		Hits         []struct {
			Document   map[string]interface{} `json:"document"`
			TextMatch  float64                `json:"text_match"`
			Highlight  map[string]interface{} `json:"highlight"`
			Highlights []struct {
				Field   string `json:"field"`
				Snippet string `json:"snippet"`
			} `json:"highlights"`
		} `json:"hits"`
		FacetCounts []struct {
			FieldName string `json:"field_name"`
			Counts    []struct {
				Value string `json:"value"`
				Count int64  `json:"count"`
			} `json:"counts"`
		} `json:"facet_counts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, errors.Internal("failed to decode typesense search", err)
	}

	result := &search.SearchResult{
		Total: raw.Found,
		Took:  time.Since(start),
		Hits:  make([]search.Hit, 0, len(raw.Hits)),
	}
	if raw.SearchTimeMS > 0 {
		result.Took = time.Duration(raw.SearchTimeMS) * time.Millisecond
	}
	var maxScore float64
	for _, h := range raw.Hits {
		id, _ := h.Document["id"].(string)
		hit := search.Hit{
			ID:     id,
			Score:  h.TextMatch,
			Source: h.Document,
		}
		if len(h.Highlights) > 0 {
			hit.Highlights = make(map[string][]string)
			for _, hl := range h.Highlights {
				hit.Highlights[hl.Field] = append(hit.Highlights[hl.Field], hl.Snippet)
			}
		}
		if hit.Score > maxScore {
			maxScore = hit.Score
		}
		result.Hits = append(result.Hits, hit)
	}
	result.MaxScore = maxScore
	if len(raw.FacetCounts) > 0 {
		result.Facets = make(map[string][]search.FacetValue)
		for _, fc := range raw.FacetCounts {
			vals := make([]search.FacetValue, 0, len(fc.Counts))
			for _, c := range fc.Counts {
				vals = append(vals, search.FacetValue{Value: c.Value, Count: c.Count})
			}
			result.Facets[fc.FieldName] = vals
		}
	}
	return result, nil
}

func formatFilter(f search.Filter) string {
	switch f.Operator {
	case search.FilterOperatorNotEquals:
		return fmt.Sprintf("%s:!=%v", f.Field, f.Value)
	case search.FilterOperatorGreaterThan:
		return fmt.Sprintf("%s:>%v", f.Field, f.Value)
	case search.FilterOperatorLessThan:
		return fmt.Sprintf("%s:<%v", f.Field, f.Value)
	case search.FilterOperatorGreaterOrEq:
		return fmt.Sprintf("%s:>=%v", f.Field, f.Value)
	case search.FilterOperatorLessOrEq:
		return fmt.Sprintf("%s:<=%v", f.Field, f.Value)
	case search.FilterOperatorIn:
		return fmt.Sprintf("%s:=[%v]", f.Field, f.Value)
	default:
		return fmt.Sprintf("%s:=%v", f.Field, f.Value)
	}
}

// Suggest uses Typesense prefix search for autocomplete.
func (e *Engine) Suggest(ctx context.Context, indexName string, query search.SuggestQuery) ([]search.Suggestion, error) {
	size := query.Size
	if size <= 0 {
		size = 5
	}
	q := search.Query{Text: query.Prefix, Size: size}
	if query.Field != "" {
		q.Fields = []string{query.Field}
	}
	res, err := e.Search(ctx, indexName, q)
	if err != nil {
		return nil, err
	}
	out := make([]search.Suggestion, 0, len(res.Hits))
	for _, h := range res.Hits {
		text := query.Prefix
		if query.Field != "" {
			if v, ok := h.Source[query.Field]; ok {
				text = fmt.Sprint(v)
			}
		} else if t, ok := h.Source["title"]; ok {
			text = fmt.Sprint(t)
		} else if id, ok := h.Source["id"]; ok {
			text = fmt.Sprint(id)
		}
		out = append(out, search.Suggestion{
			Text:    text,
			Score:   h.Score,
			Payload: map[string]interface{}{"id": h.ID},
		})
	}
	return out, nil
}

// Bulk imports documents via Typesense import API.
func (e *Engine) Bulk(ctx context.Context, indexName string, ops []search.BulkOperation) (*search.BulkResult, error) {
	if len(ops) == 0 {
		return &search.BulkResult{}, nil
	}
	var buf bytes.Buffer
	for i, op := range ops {
		if i > 0 {
			buf.WriteByte('\n')
		}
		switch op.Action {
		case search.BulkActionDelete:
			line, _ := json.Marshal(map[string]interface{}{"id": op.ID})
			buf.Write(line)
		default:
			docMap, err := toDocMap(op.Document, op.ID)
			if err != nil {
				return nil, err
			}
			line, err := json.Marshal(docMap)
			if err != nil {
				return nil, errors.Internal("failed to encode bulk document", err)
			}
			buf.Write(line)
		}
	}

	action := "upsert"
	path := "/collections/" + url.PathEscape(indexName) + "/documents/import?action=" + action
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+path, &buf)
	if err != nil {
		return nil, errors.Internal("failed to create typesense request", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	if e.apiKey != "" {
		req.Header.Set("X-TYPESENSE-API-KEY", e.apiKey)
	}
	start := time.Now()
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, errors.Internal("typesense request failed", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, errors.Internal(fmt.Sprintf("typesense bulk error: %s", string(body)), nil)
	}

	result := &search.BulkResult{Took: time.Since(start)}
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		if line == "" {
			continue
		}
		var row struct {
			Success  bool   `json:"success"`
			Error    string `json:"error"`
			ID       string `json:"id"`
			Document struct {
				ID string `json:"id"`
			} `json:"document"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, search.BulkError{Reason: err.Error()})
			continue
		}
		if row.Success {
			result.Successful++
		} else {
			result.Failed++
			id := row.ID
			if id == "" {
				id = row.Document.ID
			}
			result.Errors = append(result.Errors, search.BulkError{ID: id, Reason: row.Error})
		}
	}
	return result, nil
}

// Refresh is a no-op for Typesense (writes are immediately searchable).
func (e *Engine) Refresh(ctx context.Context, indexName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_ = indexName
	return nil
}

// Close releases resources.
func (e *Engine) Close() error { return nil }

func (e *Engine) do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Internal("failed to encode typesense body", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, e.baseURL+path, rdr)
	if err != nil {
		return nil, errors.Internal("failed to create typesense request", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if e.apiKey != "" {
		req.Header.Set("X-TYPESENSE-API-KEY", e.apiKey)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, errors.Internal("typesense request failed", err)
	}
	return resp, nil
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
		return errors.Internal(fmt.Sprintf("typesense error: %s", msg), nil)
	}
}
