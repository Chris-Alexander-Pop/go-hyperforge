package secrets

import "context"

// Client abstracts secret retrieval.
type Client interface {
	// GetSecret retrieves a secret value by key/name.
	GetSecret(ctx context.Context, key string) (string, error)

	// Close closes any underlying connections.
	Close() error
}
