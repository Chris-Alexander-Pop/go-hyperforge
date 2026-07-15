package memory

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
)

// KeyProvider is an in-memory crypto.KeyProvider for tests and local development.
// DO NOT use in production — keys should come from a real KMS.
type KeyProvider struct {
	masterKey []byte
}

// Ensure KeyProvider implements crypto.KeyProvider.
var _ crypto.KeyProvider = (*KeyProvider)(nil)

// NewKeyProvider creates an in-memory key provider.
// masterKey must be 32 bytes (AES-256).
func NewKeyProvider(masterKey []byte) (*KeyProvider, error) {
	if len(masterKey) != 32 {
		return nil, crypto.ErrInvalidKey
	}
	cp := make([]byte, len(masterKey))
	copy(cp, masterKey)
	return &KeyProvider{masterKey: cp}, nil
}

// GetKey returns the master key (keyID is ignored).
func (m *KeyProvider) GetKey(ctx context.Context, keyID string) ([]byte, error) {
	_ = ctx
	_ = keyID
	out := make([]byte, len(m.masterKey))
	copy(out, m.masterKey)
	return out, nil
}

// GenerateDataKey generates a random DEK and wraps it with the master key.
func (m *KeyProvider) GenerateDataKey(ctx context.Context) ([]byte, []byte, string, error) {
	_ = ctx
	dek, err := crypto.GenerateAES256Key()
	if err != nil {
		return nil, nil, "", err
	}

	encryptor, err := crypto.NewAESEncryptor(m.masterKey)
	if err != nil {
		return nil, nil, "", err
	}

	encryptedDEK, err := encryptor.Encrypt(dek)
	if err != nil {
		return nil, nil, "", err
	}

	return dek, encryptedDEK, "memory-key-1", nil
}

// DecryptDataKey unwraps an encrypted DEK with the master key.
func (m *KeyProvider) DecryptDataKey(ctx context.Context, encryptedKey []byte, keyID string) ([]byte, error) {
	_ = ctx
	_ = keyID
	encryptor, err := crypto.NewAESEncryptor(m.masterKey)
	if err != nil {
		return nil, err
	}
	return encryptor.Decrypt(encryptedKey)
}
