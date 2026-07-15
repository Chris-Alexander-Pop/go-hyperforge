// Package secrets provides secret management interfaces.
//
// Adapters:
//   - adapters/memory — in-process store for tests
//   - adapters/vault — HashiCorp Vault KV v2 over HTTP (token auth)
//   - adapters/awssecrets — AWS Secrets Manager
//   - adapters/gcpsecretmanager — GCP Secret Manager
//   - adapters/azurekv — Azure Key Vault secrets (Get/Set/Delete)
//
// Optional: wrap with NewEventedSecretManager for audit-friendly domain events
// (secrets.rotated / secrets.set) via pkg/events.
package secrets
