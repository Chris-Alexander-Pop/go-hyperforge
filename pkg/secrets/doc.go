/*
Package secrets provides a unified interface for secret management.

It supports multiple backends to abstract away cloud-specific implementations:
  - Memory: For local development and testing
  - AWS Secrets Manager
  - GCP Secret Manager
  - Azure Key Vault
  - HashiCorp Vault

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/secrets"
	import "github.com/chris-alexander-pop/system-design-library/pkg/secrets/adapters/memory"

	// Create a manager
	store := memory.New()

	// Store and key
	err := store.SetSecret(ctx, "db-pass", "super-secret")
	val, err := store.GetSecret(ctx, "db-pass")
*/
package secrets
