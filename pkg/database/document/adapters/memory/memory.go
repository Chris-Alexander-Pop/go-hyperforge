package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/document"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Adapter implements document.Interface with an in-memory store.
type Adapter struct {
	collections map[string][]document.Document
	mu          *concurrency.SmartRWMutex
}

// New creates a new in-memory document store.
func New() document.Interface {
	return &Adapter{
		collections: make(map[string][]document.Document),
		mu:          concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-document"}),
	}
}

// Insert adds a new document to the collection.
func (a *Adapter) Insert(ctx context.Context, collection string, doc document.Document) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.collections[collection] == nil {
		a.collections[collection] = make([]document.Document, 0)
	}
	a.collections[collection] = append(a.collections[collection], cloneDoc(doc))
	return nil
}

// Find retrieves documents matching the query.
func (a *Adapter) Find(ctx context.Context, collection string, query map[string]interface{}) ([]document.Document, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	docs, ok := a.collections[collection]
	if !ok {
		return nil, errors.NotFound("collection not found", nil)
	}

	var results []document.Document
	for _, doc := range docs {
		if matchesQuery(doc, query) {
			results = append(results, cloneDoc(doc))
		}
	}
	return results, nil
}

// Update modifies documents matching the filter.
func (a *Adapter) Update(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	docs, ok := a.collections[collection]
	if !ok {
		return errors.NotFound("collection not found", nil)
	}

	for i, doc := range docs {
		if matchesQuery(doc, filter) {
			for k, v := range update {
				docs[i][k] = v
			}
		}
	}
	return nil
}

// Delete removes documents matching the filter.
func (a *Adapter) Delete(ctx context.Context, collection string, filter map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	docs, ok := a.collections[collection]
	if !ok {
		return nil // No-op if collection doesn't exist
	}

	var remaining []document.Document
	for _, doc := range docs {
		if !matchesQuery(doc, filter) {
			remaining = append(remaining, doc)
		}
	}
	a.collections[collection] = remaining
	return nil
}

// Close clears the in-memory store.
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.collections = make(map[string][]document.Document)
	return nil
}

// matchesQuery checks if a document matches all query conditions.
func matchesQuery(doc document.Document, query map[string]interface{}) bool {
	for k, v := range query {
		if doc[k] != v {
			return false
		}
	}
	return true
}

// cloneDoc creates a shallow copy of a document.
func cloneDoc(doc document.Document) document.Document {
	clone := make(document.Document)
	for k, v := range doc {
		clone[k] = v
	}
	return clone
}
