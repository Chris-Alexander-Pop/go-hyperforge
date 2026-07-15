// Package opensearch provides a memory-backed SearchEngine stub labeled for OpenSearch.
//
// This is NOT an OpenSearch/Elasticsearch wire client. For ES-compatible HTTP
// against a real cluster, prefer adapters/elasticsearch. This package exists so
// Config.Driver="opensearch" can resolve in tests without claiming production support.
package opensearch

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search/adapters/memory"
)

// New returns a memory-backed SearchEngine for OpenSearch-shaped local use.
func New() search.SearchEngine {
	return memory.New()
}

// NewWithConfig ignores cfg and returns a memory-backed engine.
func NewWithConfig(_ search.Config) search.SearchEngine {
	return memory.New()
}
