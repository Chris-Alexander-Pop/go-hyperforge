package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Encryptor encrypts and decrypts data.
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// KeyProvider provides encryption keys for envelope encryption.
type KeyProvider interface {
	GetKey(ctx context.Context, keyID string) ([]byte, error)
	GenerateDataKey(ctx context.Context) (key []byte, encryptedKey []byte, keyID string, err error)
	DecryptDataKey(ctx context.Context, encryptedKey []byte, keyID string) ([]byte, error)
}

// =========================================================================
// AES-GCM Encryption
// =========================================================================

// AESEncryptor provides AES-GCM authenticated encryption.
type AESEncryptor struct {
	key []byte
}

// Ensure AESEncryptor implements Encryptor.
var _ Encryptor = (*AESEncryptor)(nil)

// NewAESEncryptor creates a new AES encryptor with the given key.
// Key must be 16, 24, or 32 bytes (AES-128, AES-192, AES-256).
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKey
	}
	return &AESEncryptor{key: key}, nil
}

// Encrypt encrypts plaintext using AES-GCM.
// The nonce is prepended to the ciphertext.
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, errors.New(CodeInternal, "failed to create AES cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New(CodeInternal, "failed to create GCM", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.New(CodeInternal, "failed to generate nonce", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext using AES-GCM.
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, errors.New(CodeInternal, "failed to create AES cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New(CodeInternal, "failed to create GCM", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, ErrInvalidCiphertext
	}

	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext.
func (e *AESEncryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts base64-encoded ciphertext and returns the plaintext string.
func (e *AESEncryptor) DecryptString(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// =========================================================================
// Key Generation
// =========================================================================

// GenerateKey generates a cryptographically secure random key.
func GenerateKey(length int) ([]byte, error) {
	if length <= 0 {
		return nil, ErrInvalidKey
	}
	key := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, errors.New(CodeInternal, "failed to generate key", err)
	}
	return key, nil
}

// GenerateAES256Key generates a 256-bit key for AES-256.
func GenerateAES256Key() ([]byte, error) {
	return GenerateKey(32)
}

// GenerateNonce generates a cryptographically secure nonce.
func GenerateNonce(length int) ([]byte, error) {
	return GenerateKey(length)
}
