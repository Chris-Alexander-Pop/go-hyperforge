package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// HashConfig configures password hashing.
type HashConfig struct {
	// Algorithm is the hashing algorithm to use.
	Algorithm string // "argon2id" or "bcrypt"

	// Argon2 parameters
	Argon2Time    uint32
	Argon2Memory  uint32
	Argon2Threads uint8
	Argon2KeyLen  uint32

	// Bcrypt parameters
	BcryptCost int
}

// DefaultHashConfig returns secure defaults using Argon2id.
func DefaultHashConfig() HashConfig {
	return HashConfig{
		Algorithm:     "argon2id",
		Argon2Time:    1,
		Argon2Memory:  64 * 1024, // 64 MB
		Argon2Threads: 4,
		Argon2KeyLen:  32,
		BcryptCost:    12,
	}
}

// Hasher handles secure password hashing.
type Hasher struct {
	config HashConfig
}

// NewHasher creates a new password hasher.
func NewHasher(cfg HashConfig) *Hasher {
	return &Hasher{config: cfg}
}

// Hash creates a secure hash of the password.
func (h *Hasher) Hash(password string) (string, error) {
	switch h.config.Algorithm {
	case "bcrypt":
		return h.hashBcrypt(password)
	default:
		return h.hashArgon2id(password)
	}
}

// Verify checks if a password matches a hash.
func (h *Hasher) Verify(password, hash string) (bool, error) {
	switch h.config.Algorithm {
	case "bcrypt":
		return h.verifyBcrypt(password, hash)
	default:
		return h.verifyArgon2id(password, hash)
	}
}

// Argon2id implementation
func (h *Hasher) hashArgon2id(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.config.Argon2Time,
		h.config.Argon2Memory,
		h.config.Argon2Threads,
		h.config.Argon2KeyLen,
	)

	// Encode: $argon2id$v=19$m=MEMORY,t=TIME,p=THREADS$SALT$HASH
	encoded := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		h.config.Argon2Memory,
		h.config.Argon2Time,
		h.config.Argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

func (h *Hasher) verifyArgon2id(password, encoded string) (bool, error) {
	// Parse: $argon2id$v=19$m=65536,t=1,p=4$SALT$HASH
	parts := splitArgon2Hash(encoded)
	if len(parts) != 5 {
		return false, errors.InvalidArgument("invalid argon2id hash format", nil)
	}

	var memory, time uint32
	var threads uint8

	_, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, errors.InvalidArgument("failed to parse argon2 parameters", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, errors.InvalidArgument("invalid salt encoding", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errors.InvalidArgument("invalid hash encoding", err)
	}

	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		time,
		memory,
		threads,
		uint32(len(expectedHash)),
	)

	return constantTimeCompare(expectedHash, computedHash), nil
}

func splitArgon2Hash(hash string) []string {
	// Remove leading $ and split
	if len(hash) > 0 && hash[0] == '$' {
		hash = hash[1:]
	}

	// Handle trailing $ if present to match previous loop logic where trailing empty string is ignored
	if len(hash) > 0 && hash[len(hash)-1] == '$' {
		hash = hash[:len(hash)-1]
	}

	return strings.Split(hash, "$")
}

// Bcrypt implementation
func (h *Hasher) hashBcrypt(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.config.BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *Hasher) verifyBcrypt(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// constantTimeCompare compares two byte slices in constant time.
func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
