package algolia

import (
	"context"
	"fmt"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/opt"
	algoliasearch "github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/chris-alexander-pop/system-design-library/pkg/data/search"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Engine implements search.SearchEngine using Algolia.
type Engine struct {
	client *algoliasearch.Client
	config search.Config
}

// New creates a new Algolia search engine.
func New(cfg search.Config) (*Engine, error) {
	if cfg.AlgoliaAppID == "" || cfg.AlgoliaAPIKey == "" {
		return nil, errors.InvalidArgument("Algolia app ID and API key are required", nil)
	}

	client := algoliasearch.NewClient(cfg.AlgoliaAppID, cfg.AlgoliaAPIKey)

	return &Engine{
		client: client,
		config: cfg,
	}, nil
}

func (e *Engine) CreateIndex(ctx context.Context, indexName string, mapping *search.IndexMapping) error {
	idx := e.client.InitIndex(indexName)

	// Configure index settings from mapping
	if mapping != nil && len(mapping.Fields) > 0 {
		settings := algoliasearch.Settings{}

		var searchable, filterable []string
		for field, fm := range mapping.Fields {
			if fm.Searchable {
				searchable = append(searchable, field)
			}
			if fm.Filterable || fm.Facetable {
				filterable = append(filterable, field)
			}
		}

		if len(searchable) > 0 {
			settings.SearchableAttributes = opt.SearchableAttributes(searchable...)
		}
		if len(filterable) > 0 {
			settings.AttributesForFaceting = opt.AttributesForFaceting(filterable...)
		}

		if _, err := idx.SetSettings(settings); err != nil {
			return errors.Internal("failed to configure index settings", err)
		}
	}

	return nil
}

func (e *Engine) DeleteIndex(ctx context.Context, indexName string) error {
	idx := e.client.InitIndex(indexName)
	if _, err := idx.Delete(); err != nil {
		return errors.Internal("failed to delete index", err)
	}
	return nil
}

func (e *Engine) GetIndex(ctx context.Context, indexName string) (*search.IndexInfo, error) {
	idx := e.client.InitIndex(indexName)

	// Algolia doesn't have direct index stats, so we use list indices
	indices, err := e.client.ListIndices()
	if err != nil {
		return nil, errors.Internal("failed to list indices", err)
	}

	for _, index := range indices.Items {
		if index.Name == indexName {
			return &search.IndexInfo{
				Name:     indexName,
				DocCount: index.Entries,
			}, nil
		}
	}

	// If not found in list, try to get settings to see if it exists
	if _, err := idx.GetSettings(); err != nil {
		return nil, errors.NotFound("index not found", err)
	}

	return &search.IndexInfo{
		Name: indexName,
	}, nil
}

func (e *Engine) Index(ctx context.Context, indexName, docID string, doc interface{}) error {
	idx := e.client.InitIndex(indexName)

	// Add objectID to document
	obj := map[string]interface{}{"objectID": docID}
	if docMap, ok := doc.(map[string]interface{}); ok {
		for k, v := range docMap {
			obj[k] = v
		}
	} else {
		obj["_source"] = doc
	}

	if _, err := idx.SaveObject(obj); err != nil {
		return errors.Internal("failed to index document", err)
	}

	return nil
}

func (e *Engine) Get(ctx context.Context, indexName, docID string) (*search.Hit, error) {
	idx := e.client.InitIndex(indexName)

	var obj map[string]interface{}
	if err := idx.GetObject(docID, &obj); err != nil {
		return nil, errors.NotFound("document not found", err)
	}

	return &search.Hit{
		ID:     docID,
		Score:  1.0,
		Source: obj,
	}, nil
}

func (e *Engine) Delete(ctx context.Context, indexName, docID string) error {
	idx := e.client.InitIndex(indexName)

	if _, err := idx.DeleteObject(docID); err != nil {
		return errors.Internal("failed to delete document", err)
	}

	return nil
}

func (e *Engine) Search(ctx context.Context, indexName string, query search.Query) (*search.SearchResult, error) {
	idx := e.client.InitIndex(indexName)

	start := time.Now()

	// Build search options
	var opts []interface{}

	// Pagination
	opts = append(opts, opt.Offset(query.From))
	if query.Size > 0 {
		opts = append(opts, opt.Length(query.Size))
	} else {
		opts = append(opts, opt.Length(10))
	}

	// Filters
	if len(query.Filters) > 0 {
		filterStr := e.buildFilters(query.Filters)
		opts = append(opts, opt.Filters(filterStr))
	}

	// Facets
	if len(query.Facets) > 0 {
		opts = append(opts, opt.Facets(query.Facets...))
	}

	// Highlighting
	if query.Highlight {
		opts = append(opts, opt.AttributesToHighlight("*"))
	} else {
		opts = append(opts, opt.AttributesToHighlight())
	}

	// Restrict search to specific attributes
	if len(query.Fields) > 0 {
		opts = append(opts, opt.RestrictSearchableAttributes(query.Fields...))
	}

	res, err := idx.Search(query.Text, opts...)
	if err != nil {
		return nil, errors.Internal("failed to search", err)
	}

	// Build result
	result := &search.SearchResult{
		Total:    int64(res.NbHits),
		Took:     time.Since(start),
		Hits:     make([]search.Hit, 0, len(res.Hits)),
		MaxScore: 0,
	}

	for _, hit := range res.Hits {
		h := search.Hit{
			Source: hit,
		}

		if objID, ok := hit["objectID"].(string); ok {
			h.ID = objID
		}

		// Parse highlighting
		if highlighted, ok := hit["_highlightResult"].(map[string]interface{}); ok {
			h.Highlights = make(map[string][]string)
			for field, val := range highlighted {
				if valMap, ok := val.(map[string]interface{}); ok {
					if value, ok := valMap["value"].(string); ok {
						h.Highlights[field] = []string{value}
					}
				}
			}
		}

		result.Hits = append(result.Hits, h)
	}

	// Parse facets
	if res.Facets != nil {
		result.Facets = make(map[string][]search.FacetValue)
		for field, facetMap := range res.Facets {
			var values []search.FacetValue
			for val, count := range facetMap {
				values = append(values, search.FacetValue{
					Value: val,
					Count: int64(count),
				})
			}
			result.Facets[field] = values
		}
	}

	return result, nil
}

func (e *Engine) buildFilters(filters []search.Filter) string {
	var parts []string

	for _, f := range filters {
		var filterStr string

		switch f.Operator {
		case search.FilterOperatorEquals:
			filterStr = fmt.Sprintf("%s:%v", f.Field, f.Value)
		case search.FilterOperatorNotEquals:
			filterStr = fmt.Sprintf("NOT %s:%v", f.Field, f.Value)
		case search.FilterOperatorGreaterThan:
			filterStr = fmt.Sprintf("%s > %v", f.Field, f.Value)
		case search.FilterOperatorLessThan:
			filterStr = fmt.Sprintf("%s < %v", f.Field, f.Value)
		case search.FilterOperatorGreaterOrEq:
			filterStr = fmt.Sprintf("%s >= %v", f.Field, f.Value)
		case search.FilterOperatorLessOrEq:
			filterStr = fmt.Sprintf("%s <= %v", f.Field, f.Value)
		case search.FilterOperatorIn:
			if arr, ok := f.Value.([]interface{}); ok {
				var vals []string
				for _, v := range arr {
					vals = append(vals, fmt.Sprintf("%s:%v", f.Field, v))
				}
				filterStr = fmt.Sprintf("(%s)", join(vals, " OR "))
			}
		default:
			filterStr = fmt.Sprintf("%s:%v", f.Field, f.Value)
		}

		parts = append(parts, filterStr)
	}

	return join(parts, " AND ")
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

func (e *Engine) Bulk(ctx context.Context, indexName string, ops []search.BulkOperation) (*search.BulkResult, error) {
	start := time.Now()
	result := &search.BulkResult{}

	idx := e.client.InitIndex(indexName)

	// Group operations by type
	var saveObjects []map[string]interface{}
	var deleteIDs []string

	for _, op := range ops {
		switch op.Action {
		case search.BulkActionIndex, search.BulkActionCreate, search.BulkActionUpdate:
			obj := map[string]interface{}{"objectID": op.ID}
			if docMap, ok := op.Document.(map[string]interface{}); ok {
				for k, v := range docMap {
					obj[k] = v
				}
			} else {
				obj["_source"] = op.Document
			}
			saveObjects = append(saveObjects, obj)
		case search.BulkActionDelete:
			deleteIDs = append(deleteIDs, op.ID)
		}
	}

	// Execute save operations
	if len(saveObjects) > 0 {
		if _, err := idx.SaveObjects(saveObjects); err != nil {
			result.Failed += len(saveObjects)
			result.Errors = append(result.Errors, search.BulkError{
				ID:     "batch",
				Reason: err.Error(),
			})
		} else {
			result.Successful += len(saveObjects)
		}
	}

	// Execute delete operations
	if len(deleteIDs) > 0 {
		if _, err := idx.DeleteObjects(deleteIDs); err != nil {
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
	// Algolia updates are near-real-time, no explicit refresh needed
	return nil
}

func (e *Engine) Close() error {
	return nil
}
