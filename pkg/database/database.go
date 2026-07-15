// Package database provides a unified interface for SQL and NoSQL databases.
//
// Supported backends are now split into submodules:
//   - pkg/database/sql: SQL databases (PostgreSQL, MySQL, SQLite)
//   - pkg/database/document: Document databases (MongoDB, DynamoDB)
//   - pkg/database/vector: Vector databases (Pinecone)
//   - pkg/database/timeseries: Time-series databases (InfluxDB, Timestream)
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql/adapters/postgres"
//
//	db, err := postgres.New(cfg)
//	gormDB := db.Get(ctx)
package database

// Driver enum
const (
	DriverPostgres   = "postgres"
	DriverMySQL      = "mysql"
	DriverSQLite     = "sqlite"
	DriverSQLServer  = "sqlserver"
	DriverClickHouse = "clickhouse"

	// NoSQL Drivers
	DriverMongoDB   = "mongodb"
	DriverRedis     = "redis"
	DriverCassandra = "cassandra"
	DriverDynamoDB  = "dynamodb"

	// Vector Drivers
	DriverPinecone = "pinecone"
)

// StoreType enum
type StoreType string

const (
	StoreTypeSQL      StoreType = "sql"
	StoreTypeDocument StoreType = "document"
	StoreTypeKeyValue StoreType = "key_value"
	StoreTypeVector   StoreType = "vector"
)
