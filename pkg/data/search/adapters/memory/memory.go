package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/data/search"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// document represents a stored document.
type document struct {
	id        string
	data      map[string]interface{}
	indexedAt time.Time
}

// index represents an in-memory search index.
type index struct {
	name      string
	mapping   *search.IndexMapping
	documents map[string]*document
	createdAt time.Time
}

// Engine implements an in-memory search engine for testing.
type Engine struct {
	mu      sync.RWMutex
	indexes map[string]*index
}

// New creates a new in-memory search engine.
func New() *Engine {
	return &Engine{
		indexes: make(map[string]*index),
	}
}

// NewWithConfig creates a new in-memory search engine with config (config is ignored for memory).
func NewWithConfig(_ search.Config) *Engine {
	return New()
}

func (e *Engine) CreateIndex(ctx context.Context, indexName string, mapping *search.IndexMapping) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.indexes[indexName]; exists {
		return errors.Conflict("index already exists", nil)
	}

	e.indexes[indexName] = &index{
		name:      indexName,
		mapping:   mapping,
		documents: make(map[string]*document),
		createdAt: time.Now(),
	}

	return nil
}

func (e *Engine) DeleteIndex(ctx context.Context, indexName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.indexes[indexName]; !exists {
		return errors.NotFound("index not found", nil)
	}

	delete(e.indexes, indexName)
	return nil
}

func (e *Engine) GetIndex(ctx context.Context, indexName string) (*search.IndexInfo, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	idx, exists := e.indexes[indexName]
	if !exists {
		return nil, errors.NotFound("index not found", nil)
	}

	return &search.IndexInfo{
		Name:      indexName,
		DocCount:  int64(len(idx.documents)),
		SizeBytes: 0, // Memory doesn't track size
		CreatedAt: idx.createdAt,
	}, nil
}

func (e *Engine) Index(ctx context.Context, indexName, docID string, doc interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.indexLocked(indexName, docID, doc)
}

func (e *Engine) indexLocked(indexName, docID string, doc interface{}) error {
	idx, exists := e.indexes[indexName]
	if !exists {
		// Auto-create index
		idx = &index{
			name:      indexName,
			documents: make(map[string]*document),
			createdAt: time.Now(),
		}
		e.indexes[indexName] = idx
	}

	// Convert doc to map
	data, ok := doc.(map[string]interface{})
	if !ok {
		// Try to use as-is, wrap in a map
		data = map[string]interface{}{"_source": doc}
	}

	idx.documents[docID] = &document{
		id:        docID,
		data:      data,
		indexedAt: time.Now(),
	}

	return nil
}

func (e *Engine) Get(ctx context.Context, indexName, docID string) (*search.Hit, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	idx, exists := e.indexes[indexName]
	if !exists {
		return nil, errors.NotFound("index not found", nil)
	}

	doc, exists := idx.documents[docID]
	if !exists {
		return nil, errors.NotFound("document not found", nil)
	}

	return &search.Hit{
		ID:     docID,
		Score:  1.0,
		Source: doc.data,
	}, nil
}

func (e *Engine) Delete(ctx context.Context, indexName, docID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.deleteLocked(indexName, docID)
}

func (e *Engine) deleteLocked(indexName, docID string) error {
	idx, exists := e.indexes[indexName]
	if !exists {
		return errors.NotFound("index not found", nil)
	}

	if _, exists := idx.documents[docID]; !exists {
		return errors.NotFound("document not found", nil)
	}

	delete(idx.documents, docID)
	return nil
}

func (e *Engine) Search(ctx context.Context, indexName string, query search.Query) (*search.SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	start := time.Now()

	idx, exists := e.indexes[indexName]
	if !exists {
		return nil, errors.NotFound("index not found", nil)
	}

	var hits []search.Hit

	// Simple text matching
	for docID, doc := range idx.documents {
		if e.matchesQuery(doc, query) {
			score := e.calculateScore(doc, query)
			hit := search.Hit{
				ID:     docID,
				Score:  score,
				Source: doc.data,
			}

			if query.Highlight {
				hit.Highlights = e.generateHighlights(doc, query)
			}

			hits = append(hits, hit)
		}
	}

	// Sort by score (descending)
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})

	// Apply custom sort if specified
	if len(query.Sort) > 0 {
		e.applySort(hits, query.Sort)
	}

	total := int64(len(hits))

	// Apply pagination
	from := query.From
	if from < 0 {
		from = 0
	}
	if from >= len(hits) {
		hits = []search.Hit{}
	} else {
		hits = hits[from:]
	}

	size := query.Size
	if size <= 0 {
		size = 10
	}
	if size > len(hits) {
		size = len(hits)
	}
	hits = hits[:size]

	// Calculate max score
	var maxScore float64
	if len(hits) > 0 {
		maxScore = hits[0].Score
	}

	return &search.SearchResult{
		Hits:     hits,
		Total:    total,
		MaxScore: maxScore,
		Took:     time.Since(start),
		Facets:   e.calculateFacets(idx, query),
	}, nil
}

func (e *Engine) matchesQuery(doc *document, query search.Query) bool {
	// Empty query matches all
	if query.Text == "" && len(query.Filters) == 0 {
		return true
	}

	// Check text match
	if query.Text != "" {
		textLower := strings.ToLower(query.Text)
		found := false

		fieldsToSearch := query.Fields
		if len(fieldsToSearch) == 0 {
			// Search all fields
			for _, v := range doc.data {
				if str, ok := v.(string); ok {
					if strings.Contains(strings.ToLower(str), textLower) {
						found = true
						break
					}
				}
			}
		} else {
			for _, field := range fieldsToSearch {
				if v, ok := doc.data[field]; ok {
					if str, ok := v.(string); ok {
						if strings.Contains(strings.ToLower(str), textLower) {
							found = true
							break
						}
					}
				}
			}
		}

		if !found {
			return false
		}
	}

	// Check filters
	for _, filter := range query.Filters {
		if !e.matchesFilter(doc, filter) {
			return false
		}
	}

	return true
}

func (e *Engine) matchesFilter(doc *document, filter search.Filter) bool {
	value, exists := doc.data[filter.Field]
	if !exists {
		return filter.Operator == search.FilterOperatorExists && filter.Value == false
	}

	switch filter.Operator {
	case search.FilterOperatorEquals:
		return value == filter.Value
	case search.FilterOperatorNotEquals:
		return value != filter.Value
	case search.FilterOperatorExists:
		return exists == (filter.Value == true)
	case search.FilterOperatorIn:
		if arr, ok := filter.Value.([]interface{}); ok {
			for _, v := range arr {
				if v == value {
					return true
				}
			}
		}
		return false
	default:
		// Comparison operators for numbers
		return e.compareValues(value, filter.Operator, filter.Value)
	}
}

func (e *Engine) compareValues(docValue interface{}, op search.FilterOperator, filterValue interface{}) bool {
	// Simple numeric comparison
	docNum, dok := toFloat64(docValue)
	filterNum, fok := toFloat64(filterValue)

	if !dok || !fok {
		return false
	}

	switch op {
	case search.FilterOperatorGreaterThan:
		return docNum > filterNum
	case search.FilterOperatorLessThan:
		return docNum < filterNum
	case search.FilterOperatorGreaterOrEq:
		return docNum >= filterNum
	case search.FilterOperatorLessOrEq:
		return docNum <= filterNum
	}

	return false
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	}
	return 0, false
}

func (e *Engine) calculateScore(doc *document, query search.Query) float64 {
	if query.Text == "" {
		return 1.0
	}

	// Simple TF score - count matches
	textLower := strings.ToLower(query.Text)
	score := 0.0

	for _, v := range doc.data {
		if str, ok := v.(string); ok {
			score += float64(strings.Count(strings.ToLower(str), textLower))
		}
	}

	return score + 1.0 // Minimum score of 1.0 for matches
}

func (e *Engine) generateHighlights(doc *document, query search.Query) map[string][]string {
	if query.Text == "" {
		return nil
	}

	highlights := make(map[string][]string)
	textLower := strings.ToLower(query.Text)

	for field, v := range doc.data {
		if str, ok := v.(string); ok {
			strLower := strings.ToLower(str)
			if strings.Contains(strLower, textLower) {
				// Simple highlight: add emphasis markers
				highlighted := strings.ReplaceAll(str, query.Text, "<em>"+query.Text+"</em>")
				highlights[field] = []string{highlighted}
			}
		}
	}

	return highlights
}

func (e *Engine) applySort(hits []search.Hit, sorts []search.SortOption) {
	sort.Slice(hits, func(i, j int) bool {
		for _, s := range sorts {
			vi := hits[i].Source[s.Field]
			vj := hits[j].Source[s.Field]

			cmp := compareAny(vi, vj)
			if cmp == 0 {
				continue
			}

			if s.Descending {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

func compareAny(a, b interface{}) int {
	// String comparison
	if sa, ok := a.(string); ok {
		if sb, ok := b.(string); ok {
			return strings.Compare(sa, sb)
		}
	}

	// Numeric comparison
	if na, ok := toFloat64(a); ok {
		if nb, ok := toFloat64(b); ok {
			if na < nb {
				return -1
			} else if na > nb {
				return 1
			}
			return 0
		}
	}

	return 0
}

func (e *Engine) calculateFacets(idx *index, query search.Query) map[string][]search.FacetValue {
	if len(query.Facets) == 0 {
		return nil
	}

	facets := make(map[string][]search.FacetValue)

	for _, field := range query.Facets {
		counts := make(map[interface{}]int64)

		for _, doc := range idx.documents {
			if v, ok := doc.data[field]; ok {
				counts[v]++
			}
		}

		var values []search.FacetValue
		for v, count := range counts {
			values = append(values, search.FacetValue{
				Value: v,
				Count: count,
			})
		}

		// Sort by count descending
		sort.Slice(values, func(i, j int) bool {
			return values[i].Count > values[j].Count
		})

		facets[field] = values
	}

	return facets
}

func (e *Engine) Bulk(ctx context.Context, indexName string, ops []search.BulkOperation) (*search.BulkResult, error) {
	start := time.Now()
	result := &search.BulkResult{}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, op := range ops {
		var err error

		switch op.Action {
		case search.BulkActionIndex, search.BulkActionCreate:
			err = e.indexLocked(indexName, op.ID, op.Document)
		case search.BulkActionUpdate:
			err = e.indexLocked(indexName, op.ID, op.Document)
		case search.BulkActionDelete:
			err = e.deleteLocked(indexName, op.ID)
		}

		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, search.BulkError{
				ID:     op.ID,
				Reason: err.Error(),
			})
		} else {
			result.Successful++
		}
	}

	result.Took = time.Since(start)
	return result, nil
}

func (e *Engine) Refresh(ctx context.Context, indexName string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, exists := e.indexes[indexName]; !exists {
		return errors.NotFound("index not found", nil)
	}

	// Memory store is always "refreshed"
	return nil
}

func (e *Engine) Close() error {
	return nil
}
