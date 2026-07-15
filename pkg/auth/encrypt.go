package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/chris-alexander-pop/system-design-library/pkg/security/crypto"
)

// NewAESEncryptorFromKey builds an AES-GCM encryptor from a configured key string.
//
// Accepted key forms (first match wins):
//  1. Empty → (nil, nil) — encryption disabled
//  2. Base64-decoded to 16/24/32 bytes
//  3. Hex-decoded to 16/24/32 bytes
//  4. Raw string of length 16/24/32
//  5. Otherwise SHA-256(key) as an AES-256 key (convenient for passphrases)
func NewAESEncryptorFromKey(key string) (*crypto.AESEncryptor, error) {
	if key == "" {
		return nil, nil
	}
	raw, err := parseAESKey(key)
	if err != nil {
		return nil, ErrInvalidConfigMsg("encryption key", err)
	}
	enc, err := crypto.NewAESEncryptor(raw)
	if err != nil {
		return nil, ErrInvalidConfigMsg("encryption key", err)
	}
	return enc, nil
}

func parseAESKey(key string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(key); err == nil && isAESKeyLen(len(b)) {
		return b, nil
	}
	if b, err := hex.DecodeString(key); err == nil && isAESKeyLen(len(b)) {
		return b, nil
	}
	if isAESKeyLen(len(key)) {
		return []byte(key), nil
	}
	sum := sha256.Sum256([]byte(key))
	return sum[:], nil
}

func isAESKeyLen(n int) bool {
	return n == 16 || n == 24 || n == 32
}
