// Package secrets provides secret management interfaces.
//
// Adapters:
//   - adapters/memory — in-process store for tests
//   - adapters/vault — HashiCorp Vault KV v2 over HTTP (token auth)
//
// AWS Secrets Manager, GCP Secret Manager, and Azure Key Vault remain reserved
// Provider names without in-tree adapters.
//
// Optional: wrap with NewEventedSecretManager for audit-friendly domain events
// (secrets.rotated / secrets.set) via pkg/events.
package secrets
