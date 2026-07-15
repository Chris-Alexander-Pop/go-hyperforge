// Package secrets provides secret management interfaces.
//
// The memory adapter is the only implementation shipped today.
// HashiCorp Vault and cloud secret managers are reserved Provider names —
// there is no production Vault/AWS/GCP/Azure adapter in this tree yet.
//
// Optional: wrap with NewEventedSecretManager for audit-friendly domain events
// (secrets.rotated / secrets.set) via pkg/events.
package secrets
