package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/data/search"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// Engine implements search.SearchEngine using Elasticsearch.
type Engine struct {
	client *elasticsearch.Client
	config search.Config
}

// New creates a new Elasticsearch search engine.
func New(cfg search.Config) (*Engine, error) {
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.ElasticsearchURL},
	}

	if cfg.ElasticsearchAPIKey != "" {
		esCfg.APIKey = cfg.ElasticsearchAPIKey
	} else if cfg.ElasticsearchUsername != "" {
		esCfg.Username = cfg.ElasticsearchUsername
		esCfg.Password = cfg.ElasticsearchPassword
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, errors.Internal("failed to create elasticsearch client", err)
	}

	return &Engine{
		client: client,
		config: cfg,
	}, nil
}

func (e *Engine) CreateIndex(ctx context.Context, indexName string, mapping *search.IndexMapping) error {
	body := map[string]interface{}{}

	if mapping != nil {
		// Build settings
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

		// Build mappings
		if len(mapping.Fields) > 0 {
			properties := make(map[string]interface{})
			for fieldName, fieldMapping := range mapping.Fields {
				properties[fieldName] = e.buildFieldMapping(fieldMapping)
			}
			body["mappings"] = map[string]interface{}{
				"properties": properties,
			}
		}
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return errors.Internal("failed to encode mapping", err)
	}

	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  &buf,
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return errors.Internal("failed to create index", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return e.parseError(res)
	}

	return nil
}

func (e *Engine) buildFieldMapping(fm search.FieldMapping) map[string]interface{} {
	m := map[string]interface{}{
		"type": string(fm.Type),
	}
	if fm.Analyzer != "" {
		m["analyzer"] = fm.Analyzer
	}
	if !fm.Searchable && fm.Type == search.FieldTypeText {
		m["index"] = false
	}
	return m
}

func (e *Engine) DeleteIndex(ctx context.Context, indexName string) error {
	req := esapi.IndicesDeleteRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return errors.Internal("failed to delete index", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return e.parseError(res)
	}

	return nil
}

func (e *Engine) GetIndex(ctx context.Context, indexName string) (*search.IndexInfo, error) {
	// Get index stats
	req := esapi.IndicesStatsRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, errors.Internal("failed to get index stats", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, e.parseError(res)
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

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, errors.Internal("failed to decode response", err)
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

func (e *Engine) Index(ctx context.Context, indexName, docID string, doc interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		return errors.Internal("failed to encode document", err)
	}

	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: docID,
		Body:       &buf,
		Refresh:    "false",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return errors.Internal("failed to index document", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return e.parseError(res)
	}

	return nil
}

func (e *Engine) Get(ctx context.Context, indexName, docID string) (*search.Hit, error) {
	req := esapi.GetRequest{
		Index:      indexName,
		DocumentID: docID,
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, errors.Internal("failed to get document", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, errors.NotFound("document not found", nil)
	}

	if res.IsError() {
		return nil, e.parseError(res)
	}

	var result struct {
		ID     string                 `json:"_id"`
		Source map[string]interface{} `json:"_source"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, errors.Internal("failed to decode response", err)
	}

	return &search.Hit{
		ID:     result.ID,
		Score:  1.0,
		Source: result.Source,
	}, nil
}

func (e *Engine) Delete(ctx context.Context, indexName, docID string) error {
	req := esapi.DeleteRequest{
		Index:      indexName,
		DocumentID: docID,
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return errors.Internal("failed to delete document", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return errors.NotFound("document not found", nil)
	}

	if res.IsError() {
		return e.parseError(res)
	}

	return nil
}

func (e *Engine) Search(ctx context.Context, indexName string, query search.Query) (*search.SearchResult, error) {
	// Build query body
	body := e.buildSearchQuery(query)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, errors.Internal("failed to encode query", err)
	}

	req := esapi.SearchRequest{
		Index: []string{indexName},
		Body:  &buf,
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, errors.Internal("failed to search", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, e.parseError(res)
	}

	return e.parseSearchResponse(res.Body)
}

func (e *Engine) buildSearchQuery(q search.Query) map[string]interface{} {
	body := make(map[string]interface{})

	// Pagination
	body["from"] = q.From
	if q.Size > 0 {
		body["size"] = q.Size
	} else {
		body["size"] = 10
	}

	// Build query
	var must []interface{}
	var filter []interface{}

	// Text query
	if q.Text != "" {
		if len(q.Fields) > 0 {
			must = append(must, map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  q.Text,
					"fields": q.Fields,
				},
			})
		} else {
			must = append(must, map[string]interface{}{
				"query_string": map[string]interface{}{
					"query": q.Text,
				},
			})
		}
	}

	// Filters
	for _, f := range q.Filters {
		filter = append(filter, e.buildFilter(f))
	}

	if len(must) > 0 || len(filter) > 0 {
		boolQuery := make(map[string]interface{})
		if len(must) > 0 {
			boolQuery["must"] = must
		}
		if len(filter) > 0 {
			boolQuery["filter"] = filter
		}
		body["query"] = map[string]interface{}{"bool": boolQuery}
	} else {
		body["query"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}

	// Sorting
	if len(q.Sort) > 0 {
		var sorts []interface{}
		for _, s := range q.Sort {
			order := "asc"
			if s.Descending {
				order = "desc"
			}
			sorts = append(sorts, map[string]interface{}{
				s.Field: map[string]interface{}{"order": order},
			})
		}
		body["sort"] = sorts
	}

	// Highlighting
	if q.Highlight {
		body["highlight"] = map[string]interface{}{
			"fields": map[string]interface{}{
				"*": map[string]interface{}{},
			},
		}
	}

	// Aggregations (facets)
	if len(q.Facets) > 0 {
		aggs := make(map[string]interface{})
		for _, f := range q.Facets {
			aggs[f] = map[string]interface{}{
				"terms": map[string]interface{}{
					"field": f,
				},
			}
		}
		body["aggs"] = aggs
	}

	// Min score
	if q.MinScore > 0 {
		body["min_score"] = q.MinScore
	}

	return body
}

func (e *Engine) buildFilter(f search.Filter) map[string]interface{} {
	switch f.Operator {
	case search.FilterOperatorEquals:
		return map[string]interface{}{
			"term": map[string]interface{}{f.Field: f.Value},
		}
	case search.FilterOperatorNotEquals:
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{
					"term": map[string]interface{}{f.Field: f.Value},
				},
			},
		}
	case search.FilterOperatorGreaterThan:
		return map[string]interface{}{
			"range": map[string]interface{}{f.Field: map[string]interface{}{"gt": f.Value}},
		}
	case search.FilterOperatorLessThan:
		return map[string]interface{}{
			"range": map[string]interface{}{f.Field: map[string]interface{}{"lt": f.Value}},
		}
	case search.FilterOperatorGreaterOrEq:
		return map[string]interface{}{
			"range": map[string]interface{}{f.Field: map[string]interface{}{"gte": f.Value}},
		}
	case search.FilterOperatorLessOrEq:
		return map[string]interface{}{
			"range": map[string]interface{}{f.Field: map[string]interface{}{"lte": f.Value}},
		}
	case search.FilterOperatorIn:
		return map[string]interface{}{
			"terms": map[string]interface{}{f.Field: f.Value},
		}
	case search.FilterOperatorExists:
		if f.Value == true {
			return map[string]interface{}{
				"exists": map[string]interface{}{"field": f.Field},
			}
		}
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{
					"exists": map[string]interface{}{"field": f.Field},
				},
			},
		}
	default:
		return map[string]interface{}{
			"term": map[string]interface{}{f.Field: f.Value},
		}
	}
}

func (e *Engine) parseSearchResponse(body io.Reader) (*search.SearchResult, error) {
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
			ID:         hit.ID,
			Score:      hit.Score,
			Source:     hit.Source,
			Highlights: hit.Highlight,
		})
	}

	// Parse facets
	if len(esResult.Aggregations) > 0 {
		result.Facets = make(map[string][]search.FacetValue)
		for name, agg := range esResult.Aggregations {
			var values []search.FacetValue
			for _, bucket := range agg.Buckets {
				values = append(values, search.FacetValue{
					Value: bucket.Key,
					Count: bucket.DocCount,
				})
			}
			result.Facets[name] = values
		}
	}

	return result, nil
}

func (e *Engine) Bulk(ctx context.Context, indexName string, ops []search.BulkOperation) (*search.BulkResult, error) {
	if len(ops) == 0 {
		return &search.BulkResult{}, nil
	}

	var buf bytes.Buffer

	for _, op := range ops {
		meta := map[string]interface{}{
			"_index": indexName,
			"_id":    op.ID,
		}

		switch op.Action {
		case search.BulkActionIndex:
			if err := json.NewEncoder(&buf).Encode(map[string]interface{}{"index": meta}); err != nil {
				return nil, errors.Internal("failed to encode bulk action", err)
			}
			if err := json.NewEncoder(&buf).Encode(op.Document); err != nil {
				return nil, errors.Internal("failed to encode document", err)
			}
		case search.BulkActionCreate:
			if err := json.NewEncoder(&buf).Encode(map[string]interface{}{"create": meta}); err != nil {
				return nil, errors.Internal("failed to encode bulk action", err)
			}
			if err := json.NewEncoder(&buf).Encode(op.Document); err != nil {
				return nil, errors.Internal("failed to encode document", err)
			}
		case search.BulkActionUpdate:
			if err := json.NewEncoder(&buf).Encode(map[string]interface{}{"update": meta}); err != nil {
				return nil, errors.Internal("failed to encode bulk action", err)
			}
			if err := json.NewEncoder(&buf).Encode(map[string]interface{}{"doc": op.Document}); err != nil {
				return nil, errors.Internal("failed to encode document", err)
			}
		case search.BulkActionDelete:
			if err := json.NewEncoder(&buf).Encode(map[string]interface{}{"delete": meta}); err != nil {
				return nil, errors.Internal("failed to encode bulk action", err)
			}
		}
	}

	req := esapi.BulkRequest{
		Body: &buf,
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, errors.Internal("failed to execute bulk request", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, e.parseError(res)
	}

	var bulkResp struct {
		Took   int64 `json:"took"`
		Errors bool  `json:"errors"`
		Items  []map[string]struct {
			ID     string `json:"_id"`
			Status int    `json:"status"`
			Error  *struct {
				Reason string `json:"reason"`
			} `json:"error,omitempty"`
		} `json:"items"`
	}

	if err := json.NewDecoder(res.Body).Decode(&bulkResp); err != nil {
		return nil, errors.Internal("failed to decode bulk response", err)
	}

	result := &search.BulkResult{
		Took: time.Duration(bulkResp.Took) * time.Millisecond,
	}

	for _, item := range bulkResp.Items {
		for _, action := range item {
			if action.Error != nil {
				result.Failed++
				result.Errors = append(result.Errors, search.BulkError{
					ID:     action.ID,
					Reason: action.Error.Reason,
				})
			} else {
				result.Successful++
			}
		}
	}

	return result, nil
}

func (e *Engine) Refresh(ctx context.Context, indexName string) error {
	req := esapi.IndicesRefreshRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return errors.Internal("failed to refresh index", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return e.parseError(res)
	}

	return nil
}

func (e *Engine) Close() error {
	return nil
}

func (e *Engine) parseError(res *esapi.Response) error {
	body, _ := io.ReadAll(res.Body)

	var errResp struct {
		Error struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Reason != "" {
		msg := fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Reason)

		switch {
		case strings.Contains(errResp.Error.Type, "not_found"):
			return errors.NotFound(msg, nil)
		case strings.Contains(errResp.Error.Type, "already_exists"):
			return errors.Conflict(msg, nil)
		case strings.Contains(errResp.Error.Type, "parsing"):
			return errors.InvalidArgument(msg, nil)
		default:
			return errors.Internal(msg, nil)
		}
	}

	return errors.Internal(fmt.Sprintf("elasticsearch error: %s", string(body)), nil)
}
