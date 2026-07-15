package secrets

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
)

// Config configures the Secret Manager.
type Config struct {
	// Provider selects the secrets backend.
	// Implemented: "memory", "vault" (KV v2 HTTP),
	// "aws-secrets-manager" (adapters/awssecrets),
	// "gcp-secret-manager" (adapters/gcpsecretmanager),
	// "azure-key-vault" (adapters/azurekv).
	Provider string `env:"SECURITY_SECRETS_PROVIDER" env-default:"memory" validate:"required"`
}

// DefaultConfig returns Config with package defaults.
func DefaultConfig() Config {
	return Config{Provider: security.ProviderMemory}
}

// Validate checks Config with pkg/validator.
func (c Config) Validate() error {
	if c.Provider == "" {
		return ErrInvalidArgument
	}
	if err := validator.New().ValidateStruct(context.Background(), c); err != nil {
		if errors.IsCode(err, errors.CodeInvalidArgument) {
			return err
		}
		return errors.New(CodeInvalidArgument, "invalid secrets config", err)
	}
	return nil
}

// SecretManager defines the interface for secrets management.
type SecretManager interface {
	Get(ctx context.Context, name string) (string, error)
	Set(ctx context.Context, name, value string) error

	// Rotate replaces the secret value.
	// When newValue is empty, the adapter may generate a replacement or return
	// ErrInvalidArgument. Returns the value stored after rotation.
	Rotate(ctx context.Context, name, newValue string) (string, error)
}
