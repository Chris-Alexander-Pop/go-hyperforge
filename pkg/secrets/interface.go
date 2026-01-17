// Package secrets provides a unified interface for secret management services.
//
// Supported backends:
//   - AWS Secrets Manager
//   - GCP Secret Manager
//   - Azure Key Vault
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/secrets/adapters/aws"
//
//	client := aws.New(cfg)
//	secret, err := client.GetSecret(ctx, "database-password")
package secrets

import "context"

// Client abstracts secret retrieval.
type Client interface {
	// GetSecret retrieves a secret value by key/name.
	GetSecret(ctx context.Context, key string) (string, error)

	// Close closes any underlying connections.
	Close() error
}
