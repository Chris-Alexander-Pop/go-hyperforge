package memory

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// KeyManager implements kms.KeyManager using local AES-GCM.
// WARNING: This is for testing/development only. It uses a fixed key for everything
// or generates one on startup, meaning persistence is lost on restart if generated.
type KeyManager struct {
	masterKey []byte
}

// New creates a new in-memory KMS.
// masterKeyStr should be a 32-byte base64 string. If empty, a random one is generated.
func New(masterKeyStr string) (*KeyManager, error) {
	var key []byte
	var err error

	if masterKeyStr != "" {
		key, err = base64.StdEncoding.DecodeString(masterKeyStr)
		if err != nil {
			return nil, errors.InvalidArgument("invalid master key format", err)
		}
	} else {
		key = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, errors.Internal("failed to generate random key", err)
		}
	}

	if len(key) != 32 {
		return nil, errors.InvalidArgument("master key must be 32 bytes (AES-256)", nil)
	}

	return &KeyManager{masterKey: key}, nil
}

func (m *KeyManager) Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	// In a real KMS, keyID would select distinct keys.
	// Here we use the single master key for everything, ignoring keyID (or mixing it in).

	block, err := aes.NewCipher(m.masterKey)
	if err != nil {
		return nil, errors.Internal("failed to create cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Internal("failed to create gcm", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Internal("failed to generate nonce", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (m *KeyManager) Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.masterKey)
	if err != nil {
		return nil, errors.Internal("failed to create cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Internal("failed to create gcm", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.InvalidArgument("ciphertext too short", nil)
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Internal("failed to decrypt", err)
	}

	return plaintext, nil
}
