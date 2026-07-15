package crypto

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// EnvelopeEncryption implements the envelope encryption pattern.
// Data is encrypted with a DEK (Data Encryption Key), and the DEK
// is encrypted with a KEK (Key Encryption Key) from a KeyProvider.
//
// This pattern allows:
//   - Fast local encryption with AES
//   - Secure key management via a KMS-backed KeyProvider
//   - Key rotation without re-encrypting all data
//
// For development, use crypto/adapters/memory.NewKeyProvider.
// Cloud KMS adapters are not shipped yet.
type EnvelopeEncryption struct {
	kms KeyProvider
}

// EnvelopePayload contains encrypted data and its encrypted DEK.
type EnvelopePayload struct {
	EncryptedData string `json:"encrypted_data"` // Base64-encoded encrypted data
	EncryptedDEK  string `json:"encrypted_dek"`  // Base64-encoded KMS-encrypted DEK
	KeyID         string `json:"key_id"`         // KMS key ID used
	Algorithm     string `json:"algorithm"`      // Encryption algorithm
}

// NewEnvelopeEncryption creates a new envelope encryptor.
func NewEnvelopeEncryption(kms KeyProvider) *EnvelopeEncryption {
	return &EnvelopeEncryption{kms: kms}
}

// Encrypt encrypts data using envelope encryption.
// 1. Generate a DEK from KMS
// 2. Encrypt data with DEK using AES-GCM
// 3. Return encrypted data + encrypted DEK
func (e *EnvelopeEncryption) Encrypt(ctx context.Context, plaintext []byte) (*EnvelopePayload, error) {
	if e.kms == nil {
		return nil, errors.New(CodeInvalidKey, "key provider is required", nil)
	}

	dek, encryptedDEK, keyID, err := e.kms.GenerateDataKey(ctx)
	if err != nil {
		return nil, err
	}

	encryptor, err := NewAESEncryptor(dek)
	if err != nil {
		return nil, err
	}

	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}

	for i := range dek {
		dek[i] = 0
	}

	return &EnvelopePayload{
		EncryptedData: base64.StdEncoding.EncodeToString(ciphertext),
		EncryptedDEK:  base64.StdEncoding.EncodeToString(encryptedDEK),
		KeyID:         keyID,
		Algorithm:     "AES-256-GCM",
	}, nil
}

// Decrypt decrypts envelope-encrypted data.
// 1. Decrypt DEK using KMS
// 2. Decrypt data with DEK
func (e *EnvelopeEncryption) Decrypt(ctx context.Context, payload *EnvelopePayload) ([]byte, error) {
	if e.kms == nil {
		return nil, errors.New(CodeInvalidKey, "key provider is required", nil)
	}
	if payload == nil {
		return nil, ErrInvalidCiphertext
	}

	encryptedDEK, err := base64.StdEncoding.DecodeString(payload.EncryptedDEK)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	dek, err := e.kms.DecryptDataKey(ctx, encryptedDEK, payload.KeyID)
	if err != nil {
		return nil, err
	}
	defer func() {
		for i := range dek {
			dek[i] = 0
		}
	}()

	ciphertext, err := base64.StdEncoding.DecodeString(payload.EncryptedData)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	encryptor, err := NewAESEncryptor(dek)
	if err != nil {
		return nil, err
	}

	return encryptor.Decrypt(ciphertext)
}

// EncryptToJSON encrypts and returns JSON-serialized envelope payload.
func (e *EnvelopeEncryption) EncryptToJSON(ctx context.Context, plaintext []byte) ([]byte, error) {
	payload, err := e.Encrypt(ctx, plaintext)
	if err != nil {
		return nil, err
	}
	return json.Marshal(payload)
}

// DecryptFromJSON decrypts JSON-serialized envelope payload.
func (e *EnvelopeEncryption) DecryptFromJSON(ctx context.Context, data []byte) ([]byte, error) {
	var payload EnvelopePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, ErrInvalidCiphertext
	}
	return e.Decrypt(ctx, &payload)
}
