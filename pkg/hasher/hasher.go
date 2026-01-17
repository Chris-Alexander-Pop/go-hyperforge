// Package hasher provides password hashing using Argon2id.
//
// Argon2id is the recommended algorithm for password hashing,
// providing protection against both GPU and side-channel attacks.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/hasher"
//
//	h := hasher.New()
//	hash, err := h.Hash("password123")
//	ok := h.Verify("password123", hash)
package hasher

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Hasher struct{}

func New() *Hasher {
	return &Hasher{}
}

func (h *Hasher) Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("%s$%s", b64Salt, b64Hash), nil
}

func (h *Hasher) Verify(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return parts[1] == b64Hash
}
