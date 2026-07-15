package crypto

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

const (
	CodeInvalidKey        = "CRYPTO_INVALID_KEY"
	CodeInvalidCiphertext = "CRYPTO_INVALID_CIPHERTEXT"
	CodeDecryptionFailed  = "CRYPTO_DECRYPTION_FAILED"
	CodeInternal          = "CRYPTO_INTERNAL"
)

var (
	// ErrInvalidKey is returned when a key length or format is unsupported.
	ErrInvalidKey = errors.New(CodeInvalidKey, "invalid encryption key", nil)

	// ErrInvalidCiphertext is returned when ciphertext is truncated or malformed.
	ErrInvalidCiphertext = errors.New(CodeInvalidCiphertext, "invalid ciphertext", nil)

	// ErrDecryptionFailed is returned when authenticated decryption fails.
	ErrDecryptionFailed = errors.New(CodeDecryptionFailed, "decryption failed", nil)

	// ErrInternal wraps unexpected crypto failures (RNG, cipher init).
	ErrInternal = errors.New(CodeInternal, "internal crypto error", nil)
)
