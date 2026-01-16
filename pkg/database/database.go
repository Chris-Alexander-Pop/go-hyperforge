package database

import (
	"context"

	"gorm.io/gorm"
)

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

// Config holds configuration for a single database connection
type Config struct {
	Driver string    `env:"DB_DRIVER" env-default:"postgres"`
	Type   StoreType `env:"DB_TYPE" env-default:"sql"` // sql, document, key_value, vector

	// Common connection params
	Host     string `env:"DB_HOST" env-default:"localhost"`
	Port     string `env:"DB_PORT" env-default:"5432"`
	User     string `env:"DB_USER" env-default:"postgres"`
	Password string `env:"DB_PASSWORD" env-default:"postgres"`
	Name     string `env:"DB_NAME" env-default:"postgres"` // DB Name, Keyspace, or Index Name

	// Cloud / SSL Configuration
	SSLMode     string `env:"DB_SSL_MODE" env-default:"disable"` // disable, require, verify-ca, verify-full
	SSLCert     string `env:"DB_SSL_CERT"`                       // Path to client cert
	SSLKey      string `env:"DB_SSL_KEY"`                        // Path to client key
	SSLRootCert string `env:"DB_SSL_ROOT_CERT"`                  // Path to CA cert
	AuthToken   string `env:"DB_AUTH_TOKEN"`                     // For IAM authentication (e.g. AWS RDS)

	// NoSQL / Vector Specifics
	Region      string `env:"DB_REGION"`      // AWS Region for DynamoDB/Pinecone
	APIKey      string `env:"DB_API_KEY"`     // Pinecone / Cloud API Key
	Environment string `env:"DB_ENVIRONMENT"` // Pinecone Environment (e.g. us-west1-gcp)
	ProjectID   string `env:"DB_PROJECT_ID"`  // GCP Project ID / Firestore
}

// ManagerConfig holds configuration for the entire database layer, including shards
type ManagerConfig struct {
	Primary          Config
	Shards           map[string]Config // Keyed by shard ID (e.g., "shard-1", "shard-2")
	ShardingStrategy string            `env:"DB_SHARDING_STRATEGY" env-default:"manual"` // manual, consistent_hash
}

// DB defines the interface for interacting with the database layer
type DB interface {
	// SQL Accessor
	Get(ctx context.Context) *gorm.DB
	GetShard(ctx context.Context, key string) (*gorm.DB, error)

	// Generic/NoSQL Accessors (Type assertion required by caller)
	GetDocument(ctx context.Context) interface{} // Returns *mongo.Database or *dynamodb.Client
	GetKV(ctx context.Context) interface{}       // Returns *redis.Client

	// Vector Accessor (Unified Interface)
	GetVector(ctx context.Context) interface{} // Returns vector.Store

	// Close closes all connections
	Close() error
}
