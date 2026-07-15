package bigdata

import (
	"context"
)

// Config configures the bigdata package.
type Config struct {
	// Adapter specifies the bigdata backend (bigquery, redshift, memory).
	Adapter string `env:"BIGDATA_ADAPTER" env-default:"memory"`
}

// Client is a generic interface for executing analytical queries.
// It abstracts data warehouses such as BigQuery, Redshift, and Snowflake.
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
