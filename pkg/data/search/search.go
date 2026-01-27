// Package search provides a unified interface for full-text search engines.
//
// Supported backends:
//   - Memory: In-memory search for testing
//   - Elasticsearch: Elasticsearch/OpenSearch adapter
//   - Meilisearch: Meilisearch adapter
//   - Algolia: Algolia search adapter
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/data/search/adapters/memory"
//
//	engine := memory.New()
//	err := engine.Index(ctx, "products", "123", product)
//	result, err := engine.Search(ctx, "products", search.Query{Text: "laptop"})
package search

import (
	"context"
	"time"
)

// Driver constants for search backends.
const (
	DriverMemory        = "memory"
	DriverElasticsearch = "elasticsearch"
	DriverOpenSearch    = "opensearch"
	DriverMeilisearch   = "meilisearch"
	DriverAlgolia       = "algolia"
	DriverTypesense     = "typesense"
)

// Config holds configuration for search engines.
type Config struct {
	// Driver specifies the search backend.
	Driver string `env:"SEARCH_DRIVER" env-default:"memory"`

	// Elasticsearch/OpenSearch specific
	ElasticsearchURL      string `env:"SEARCH_ES_URL" env-default:"http://localhost:9200"`
	ElasticsearchUsername string `env:"SEARCH_ES_USERNAME"`
	ElasticsearchPassword string `env:"SEARCH_ES_PASSWORD"`
	ElasticsearchAPIKey   string `env:"SEARCH_ES_API_KEY"`

	// Meilisearch specific
	MeilisearchURL    string `env:"SEARCH_MEILI_URL" env-default:"http://localhost:7700"`
	MeilisearchAPIKey string `env:"SEARCH_MEILI_API_KEY"`

	// Algolia specific
	AlgoliaAppID     string `env:"SEARCH_ALGOLIA_APP_ID"`
	AlgoliaAPIKey    string `env:"SEARCH_ALGOLIA_API_KEY"`
	AlgoliaSearchKey string `env:"SEARCH_ALGOLIA_SEARCH_KEY"`

	// Common options
	Timeout       time.Duration `env:"SEARCH_TIMEOUT" env-default:"30s"`
	MaxRetries    int           `env:"SEARCH_MAX_RETRIES" env-default:"3"`
	BulkBatchSize int           `env:"SEARCH_BULK_BATCH_SIZE" env-default:"1000"`
}

// FieldType represents the type of a field in the index mapping.
type FieldType string

const (
	FieldTypeText     FieldType = "text"
	FieldTypeKeyword  FieldType = "keyword"
	FieldTypeInteger  FieldType = "integer"
	FieldTypeLong     FieldType = "long"
	FieldTypeFloat    FieldType = "float"
	FieldTypeDouble   FieldType = "double"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeDate     FieldType = "date"
	FieldTypeGeoPoint FieldType = "geo_point"
	FieldTypeNested   FieldType = "nested"
	FieldTypeObject   FieldType = "object"
)

// FieldMapping defines how a field should be indexed.
type FieldMapping struct {
	// Type is the field data type.
	Type FieldType

	// Analyzer is the text analyzer (for text fields).
	Analyzer string

	// Searchable indicates if the field is searchable.
	Searchable bool

	// Filterable indicates if the field can be used in filters.
	Filterable bool

	// Sortable indicates if the field can be used for sorting.
	Sortable bool

	// Facetable indicates if the field can be used for faceting.
	Facetable bool
}

// IndexMapping defines the schema for an index.
type IndexMapping struct {
	// Fields maps field names to their mappings.
	Fields map[string]FieldMapping

	// Settings contains index-level settings.
	Settings IndexSettings
}

// IndexSettings contains index configuration.
type IndexSettings struct {
	// NumberOfShards is the number of primary shards.
	NumberOfShards int

	// NumberOfReplicas is the number of replica shards.
	NumberOfReplicas int

	// RefreshInterval is how often to refresh the index.
	RefreshInterval time.Duration
}

// Query represents a search query.
type Query struct {
	// Text is the main search text.
	Text string

	// Fields limits the search to specific fields.
	Fields []string

	// Filters are field-value filters to apply.
	Filters []Filter

	// Sort specifies the sort order.
	Sort []SortOption

	// From is the starting offset for pagination.
	From int

	// Size is the number of results to return.
	Size int

	// Highlight enables result highlighting.
	Highlight bool

	// Facets specifies fields to aggregate.
	Facets []string

	// MinScore filters out results below this score.
	MinScore float64
}

// Filter represents a filter condition.
type Filter struct {
	// Field is the field to filter on.
	Field string

	// Operator is the comparison operator.
	Operator FilterOperator

	// Value is the value to compare against.
	Value interface{}
}

// FilterOperator is the type of filter comparison.
type FilterOperator string

const (
	FilterOperatorEquals      FilterOperator = "eq"
	FilterOperatorNotEquals   FilterOperator = "ne"
	FilterOperatorGreaterThan FilterOperator = "gt"
	FilterOperatorLessThan    FilterOperator = "lt"
	FilterOperatorGreaterOrEq FilterOperator = "gte"
	FilterOperatorLessOrEq    FilterOperator = "lte"
	FilterOperatorIn          FilterOperator = "in"
	FilterOperatorNotIn       FilterOperator = "nin"
	FilterOperatorExists      FilterOperator = "exists"
	FilterOperatorRange       FilterOperator = "range"
)

// SortOption specifies a sort field and order.
type SortOption struct {
	// Field is the field to sort by.
	Field string

	// Descending sorts in descending order if true.
	Descending bool
}

// SearchResult contains the search results.
type SearchResult struct {
	// Hits are the matching documents.
	Hits []Hit

	// Total is the total number of matching documents.
	Total int64

	// MaxScore is the highest relevance score.
	MaxScore float64

	// Took is how long the search took.
	Took time.Duration

	// Facets contains aggregation results.
	Facets map[string][]FacetValue
}

// Hit represents a single search result.
type Hit struct {
	// ID is the document ID.
	ID string

	// Score is the relevance score.
	Score float64

	// Source is the document data.
	Source map[string]interface{}

	// Highlights contains highlighted field snippets.
	Highlights map[string][]string
}

// FacetValue represents a single facet bucket.
type FacetValue struct {
	// Value is the facet value.
	Value interface{}

	// Count is the number of documents with this value.
	Count int64
}

// BulkOperation represents a single bulk operation.
type BulkOperation struct {
	// Action is the operation type.
	Action BulkAction

	// ID is the document ID.
	ID string

	// Document is the document data (for index/update).
	Document interface{}
}

// BulkAction is the type of bulk operation.
type BulkAction string

const (
	BulkActionIndex  BulkAction = "index"
	BulkActionCreate BulkAction = "create"
	BulkActionUpdate BulkAction = "update"
	BulkActionDelete BulkAction = "delete"
)

// BulkResult contains the result of a bulk operation.
type BulkResult struct {
	// Took is how long the operation took.
	Took time.Duration

	// Successful is the number of successful operations.
	Successful int

	// Failed is the number of failed operations.
	Failed int

	// Errors lists any errors that occurred.
	Errors []BulkError
}

// BulkError represents an error in a bulk operation.
type BulkError struct {
	// ID is the document ID that failed.
	ID string

	// Reason is the error message.
	Reason string
}

// IndexInfo contains information about an index.
type IndexInfo struct {
	// Name is the index name.
	Name string

	// DocCount is the number of documents.
	DocCount int64

	// SizeBytes is the index size in bytes.
	SizeBytes int64

	// CreatedAt is when the index was created.
	CreatedAt time.Time
}

// SearchEngine defines the interface for full-text search operations.
type SearchEngine interface {
	// CreateIndex creates a new search index with the given mapping.
	CreateIndex(ctx context.Context, indexName string, mapping *IndexMapping) error

	// DeleteIndex deletes an index and all its documents.
	DeleteIndex(ctx context.Context, indexName string) error

	// GetIndex returns information about an index.
	GetIndex(ctx context.Context, indexName string) (*IndexInfo, error)

	// Index adds or updates a document in the index.
	Index(ctx context.Context, indexName, docID string, doc interface{}) error

	// Get retrieves a document by ID.
	Get(ctx context.Context, indexName, docID string) (*Hit, error)

	// Delete removes a document from the index.
	Delete(ctx context.Context, indexName, docID string) error

	// Search performs a search query.
	Search(ctx context.Context, indexName string, query Query) (*SearchResult, error)

	// Bulk performs multiple operations in a single request.
	Bulk(ctx context.Context, indexName string, ops []BulkOperation) (*BulkResult, error)

	// Refresh forces a refresh of the index, making all operations visible.
	Refresh(ctx context.Context, indexName string) error

	// Close releases resources.
	Close() error
}
