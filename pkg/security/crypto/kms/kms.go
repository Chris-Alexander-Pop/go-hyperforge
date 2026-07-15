package kms

import (
	"context"
)

// Config configures the KMS.
type Config struct {
	// Provider selects the KMS backend.
	// Implemented: "memory". Reserved (no adapter): "aws-kms", "gcp-kms", "azure-kms", "vault".
	Provider string `env:"SECURITY_KMS_PROVIDER" env-default:"memory"`
}

// KeyManager defines the interface for key management operations.
type KeyManager interface {
	Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
}
