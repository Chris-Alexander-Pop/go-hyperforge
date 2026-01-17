package bigdata

import (
	"context"
)

// Client is a generic interface for executing analytical queries.
// It abstracts Data Warehouses like BigQuery, Redshift, Snowflake, etc.
type Client interface {
	// Query executes a SQL query and returns the results.
	// The result type is generic, often []map[string]interface{}.
	Query(ctx context.Context, query string, args ...interface{}) (*Result, error)

	// Close closes the connection.
	Close() error
}

// Result represents the outcome of a query.
type Result struct {
	Rows     []map[string]interface{}
	Metadata map[string]interface{}
}
