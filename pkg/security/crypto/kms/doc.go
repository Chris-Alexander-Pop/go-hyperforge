// Package kms provides Key Management Service interfaces.
//
// Adapters:
//   - adapters/memory — local AES-GCM (dev/test only)
//   - adapters/awskms — AWS KMS Encrypt/Decrypt (SDK client or injectable API)
//
// GCP KMS and Azure Key Vault remain reserved Provider names without adapters.
package kms
