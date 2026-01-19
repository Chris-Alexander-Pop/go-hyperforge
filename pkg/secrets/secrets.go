package secrets

import "context"

// Config holds configuration for secret managers.
type Config struct {
	// Provider specifies the backend: "memory", "aws", "gcp", "azure", "vault".
	Provider string `env:"SECRETS_PROVIDER" env-default:"memory"`

	// Region is the cloud region (AWS/GCP).
	Region string `env:"SECRETS_REGION"`

	// VaultAddress is the HashiCorp Vault address.
	VaultAddress string `env:"VAULT_ADDR"`
}

// Manager abstracts secret retrieval and management.
type Manager interface {
	// GetSecret retrieves a secret value by key/name.
	GetSecret(ctx context.Context, key string) (string, error)

	// SetSecret stores a secret value.
	SetSecret(ctx context.Context, key string, value string) error

	// DeleteSecret removes a secret.
	DeleteSecret(ctx context.Context, key string) error

	// Close closes any underlying connections.
	Close() error
}
