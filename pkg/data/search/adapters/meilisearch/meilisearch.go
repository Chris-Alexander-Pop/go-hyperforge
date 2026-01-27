package meilisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/data/search"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	meili "github.com/meilisearch/meilisearch-go"
)

// Engine implements search.SearchEngine using Meilisearch.
type Engine struct {
	client meili.ServiceManager
	config search.Config
}

// New creates a new Meilisearch search engine.
func New(cfg search.Config) (*Engine, error) {
	client := meili.New(cfg.MeilisearchURL, meili.WithAPIKey(cfg.MeilisearchAPIKey))

	return &Engine{
		client: client,
		config: cfg,
	}, nil
}

func (e *Engine) CreateIndex(ctx context.Context, indexName string, mapping *search.IndexMapping) error {
	_, err := e.client.CreateIndex(&meili.IndexConfig{
		Uid:        indexName,
		PrimaryKey: "id",
	})
	if err != nil {
		return errors.Internal("failed to create index", err)
	}

	// Configure searchable and filterable attributes from mapping
	if mapping != nil && len(mapping.Fields) > 0 {
		var searchable, sortable []string
		var filterable []string

		for field, fm := range mapping.Fields {
			if fm.Searchable {
				searchable = append(searchable, field)
			}
			if fm.Filterable {
				filterable = append(filterable, field)
			}
			if fm.Sortable {
				sortable = append(sortable, field)
			}
		}

		idx := e.client.Index(indexName)
		if len(searchable) > 0 {
			if _, err := idx.UpdateSearchableAttributes(&searchable); err != nil {
				return errors.Internal("failed to update searchable attributes", err)
			}
		}
		if len(filterable) > 0 {
			// Convert []string to []interface{} as required by SDK
			filterableIface := make([]interface{}, len(filterable))
			for i, v := range filterable {
				filterableIface[i] = v
			}
			if _, err := idx.UpdateFilterableAttributes(&filterableIface); err != nil {
				return errors.Internal("failed to update filterable attributes", err)
			}
		}
		if len(sortable) > 0 {
			if _, err := idx.UpdateSortableAttributes(&sortable); err != nil {
				return errors.Internal("failed to update sortable attributes", err)
			}
		}
	}

	return nil
}

func (e *Engine) DeleteIndex(ctx context.Context, indexName string) error {
	_, err := e.client.DeleteIndex(indexName)
	if err != nil {
		return errors.Internal("failed to delete index", err)
	}
	return nil
}

func (e *Engine) GetIndex(ctx context.Context, indexName string) (*search.IndexInfo, error) {
	idx := e.client.Index(indexName)

	stats, err := idx.GetStats()
	if err != nil {
		return nil, errors.Internal("failed to get index stats", err)
	}

	return &search.IndexInfo{
		Name:     indexName,
		DocCount: stats.NumberOfDocuments,
	}, nil
}

func (e *Engine) Index(ctx context.Context, indexName, docID string, doc interface{}) error {
	idx := e.client.Index(indexName)

	// Ensure document has an ID
	docMap, ok := doc.(map[string]interface{})
	if !ok {
		docMap = map[string]interface{}{"_source": doc}
	}
	docMap["id"] = docID

	_, err := idx.AddDocuments([]map[string]interface{}{docMap}, nil)
	if err != nil {
		return errors.Internal("failed to index document", err)
	}

	return nil
}

func (e *Engine) Get(ctx context.Context, indexName, docID string) (*search.Hit, error) {
	idx := e.client.Index(indexName)

	var doc map[string]interface{}
	if err := idx.GetDocument(docID, nil, &doc); err != nil {
		return nil, errors.Internal("failed to get document", err)
	}

	return &search.Hit{
		ID:     docID,
		Score:  1.0,
		Source: doc,
	}, nil
}

func (e *Engine) Delete(ctx context.Context, indexName, docID string) error {
	idx := e.client.Index(indexName)

	_, err := idx.DeleteDocument(docID, nil)
	if err != nil {
		return errors.Internal("failed to delete document", err)
	}

	return nil
}

func (e *Engine) Search(ctx context.Context, indexName string, query search.Query) (*search.SearchResult, error) {
	idx := e.client.Index(indexName)

	start := time.Now()

	// Build search request
	searchReq := &meili.SearchRequest{
		Query:  query.Text,
		Offset: int64(query.From),
		Limit:  int64(query.Size),
	}

	if searchReq.Limit == 0 {
		searchReq.Limit = 10
	}

	// Build filters
	if len(query.Filters) > 0 {
		searchReq.Filter = e.buildFilters(query.Filters)
	}

	// Build sort
	if len(query.Sort) > 0 {
		var sorts []string
		for _, s := range query.Sort {
			order := "asc"
			if s.Descending {
				order = "desc"
			}
			sorts = append(sorts, fmt.Sprintf("%s:%s", s.Field, order))
		}
		searchReq.Sort = sorts
	}

	// Facets
	if len(query.Facets) > 0 {
		searchReq.Facets = query.Facets
	}

	// Highlighting
	if query.Highlight {
		searchReq.AttributesToHighlight = []string{"*"}
	}

	resp, err := idx.Search(query.Text, searchReq)
	if err != nil {
		return nil, errors.Internal("failed to search", err)
	}

	// Build result
	result := &search.SearchResult{
		Total: resp.EstimatedTotalHits,
		Took:  time.Since(start),
		Hits:  make([]search.Hit, 0, len(resp.Hits)),
	}

	for _, hit := range resp.Hits {
		// Decode the hit into a map
		var docMap map[string]interface{}
		if err := hit.DecodeInto(&docMap); err != nil {
			continue
		}

		h := search.Hit{
			Source: docMap,
		}

		if id, ok := docMap["id"].(string); ok {
			h.ID = id
		}

		// Parse highlighting from formatted field
		if formatted, ok := docMap["_formatted"].(map[string]interface{}); ok {
			h.Highlights = make(map[string][]string)
			for k, v := range formatted {
				if str, ok := v.(string); ok {
					h.Highlights[k] = []string{str}
				}
			}
		}

		result.Hits = append(result.Hits, h)
	}

	return result, nil
}

func (e *Engine) buildFilters(filters []search.Filter) interface{} {
	if len(filters) == 0 {
		return nil
	}

	var parts []string
	for _, f := range filters {
		switch f.Operator {
		case search.FilterOperatorEquals:
			parts = append(parts, fmt.Sprintf("%s = %v", f.Field, f.Value))
		case search.FilterOperatorNotEquals:
			parts = append(parts, fmt.Sprintf("%s != %v", f.Field, f.Value))
		case search.FilterOperatorGreaterThan:
			parts = append(parts, fmt.Sprintf("%s > %v", f.Field, f.Value))
		case search.FilterOperatorLessThan:
			parts = append(parts, fmt.Sprintf("%s < %v", f.Field, f.Value))
		case search.FilterOperatorGreaterOrEq:
			parts = append(parts, fmt.Sprintf("%s >= %v", f.Field, f.Value))
		case search.FilterOperatorLessOrEq:
			parts = append(parts, fmt.Sprintf("%s <= %v", f.Field, f.Value))
		case search.FilterOperatorExists:
			if f.Value == true {
				parts = append(parts, fmt.Sprintf("%s EXISTS", f.Field))
			} else {
				parts = append(parts, fmt.Sprintf("%s NOT EXISTS", f.Field))
			}
		}
	}

	if len(parts) == 1 {
		return parts[0]
	}
	return parts
}

func (e *Engine) Bulk(ctx context.Context, indexName string, ops []search.BulkOperation) (*search.BulkResult, error) {
	start := time.Now()
	result := &search.BulkResult{}

	idx := e.client.Index(indexName)

	// Group operations by type
	var indexDocs []map[string]interface{}
	var deleteIDs []string

	for _, op := range ops {
		switch op.Action {
		case search.BulkActionIndex, search.BulkActionCreate, search.BulkActionUpdate:
			docMap, ok := op.Document.(map[string]interface{})
			if !ok {
				docMap = map[string]interface{}{"_source": op.Document}
			}
			docMap["id"] = op.ID
			indexDocs = append(indexDocs, docMap)
		case search.BulkActionDelete:
			deleteIDs = append(deleteIDs, op.ID)
		}
	}

	// Execute index operations
	if len(indexDocs) > 0 {
		_, err := idx.AddDocuments(indexDocs, nil)
		if err != nil {
			result.Failed += len(indexDocs)
			result.Errors = append(result.Errors, search.BulkError{
				ID:     "batch",
				Reason: err.Error(),
			})
		} else {
			result.Successful += len(indexDocs)
		}
	}

	// Execute delete operations
	if len(deleteIDs) > 0 {
		_, err := idx.DeleteDocuments(deleteIDs, nil)
		if err != nil {
			result.Failed += len(deleteIDs)
			result.Errors = append(result.Errors, search.BulkError{
				ID:     "batch-delete",
				Reason: err.Error(),
			})
		} else {
			result.Successful += len(deleteIDs)
		}
	}

	result.Took = time.Since(start)
	return result, nil
}

func (e *Engine) Refresh(ctx context.Context, indexName string) error {
	// Meilisearch handles this automatically
	return nil
}

func (e *Engine) Close() error {
	return nil
}

// decodeRawMessage decodes a json.RawMessage into the given type.
func decodeRawMessage(raw json.RawMessage, v interface{}) error {
	return json.Unmarshal(raw, v)
}
