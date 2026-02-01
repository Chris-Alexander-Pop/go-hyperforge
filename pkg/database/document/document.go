package document

import (
	"context"
)

// Document is a flexible key-value map representing a document.
type Document map[string]interface{}

// Config holds configuration for the document database.
type Config struct {
	// Driver specifies the document backend: "dynamodb", "cosmosdb", "firestore".
	Driver string `env:"DOC_DRIVER" env-default:"dynamodb"`

	// Host/Endpoint for the database.
	Host string `env:"DOC_HOST"`

	// Port for the database (optional, mostly for local emulators).
	Port int `env:"DOC_PORT"`

	// Region (AWS/Azure).
	Region string `env:"DOC_REGION"`

	// ProjectID (GCP).
	ProjectID string `env:"DOC_PROJECT_ID"`

	// Database/Table Name prefix or keyspace.
	Database string `env:"DOC_DATABASE" env-default:"default"`

	// Auth credentials.
	User      string `env:"DOC_USER"`
	Password  string `env:"DOC_PASSWORD"`
	AuthToken string `env:"DOC_AUTH_TOKEN"`

	// ConsistencyLevel: eventual, strong, session (backend specific).
	ConsistencyLevel string `env:"DOC_CONSISTENCY" env-default:"strong"`

	// Connection Pool Settings
	MaxOpenConns int `env:"DOC_MAX_OPEN_CONNS" env-default:"10"`
	MaxIdleConns int `env:"DOC_MAX_IDLE_CONNS" env-default:"5"`

	// TLS Settings
	UseTLS             bool   `env:"DOC_USE_TLS"`
	InsecureSkipVerify bool   `env:"DOC_INSECURE_SKIP_VERIFY"`
	CAPath             string `env:"DOC_CA_PATH"`
	CertPath           string `env:"DOC_CERT_PATH"`
	KeyPath            string `env:"DOC_KEY_PATH"`
}

// Interface defines the standard document database methods.
type Interface interface {
	// Insert adds a new document to the collection.
	Insert(ctx context.Context, collection string, doc Document) error

	// Find retrieves documents matching the query.
	Find(ctx context.Context, collection string, query map[string]interface{}) ([]Document, error)

	// Update modifies documents matching the filter.
	Update(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) error

	// Delete removes documents matching the filter.
	Delete(ctx context.Context, collection string, filter map[string]interface{}) error

	// Close releases resources.
	Close() error
}
