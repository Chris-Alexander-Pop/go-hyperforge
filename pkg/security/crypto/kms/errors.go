package kms

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

const (
	CodeInvalidArgument = "KMS_INVALID_ARGUMENT"
	CodeEncryptFailed   = "KMS_ENCRYPT_FAILED"
	CodeDecryptFailed   = "KMS_DECRYPT_FAILED"
	CodeUnavailable     = "KMS_UNAVAILABLE"
	CodeNotSupported    = "KMS_NOT_SUPPORTED"
)

var (
	// ErrInvalidArgument is returned when key material or ciphertext is invalid.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid kms argument", nil)

	// ErrEncryptFailed is returned when encryption cannot be completed.
	ErrEncryptFailed = errors.New(CodeEncryptFailed, "kms encrypt failed", nil)

	// ErrDecryptFailed is returned when decryption cannot be completed.
	ErrDecryptFailed = errors.New(CodeDecryptFailed, "kms decrypt failed", nil)

	// ErrUnavailable is returned when a remote KMS is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "kms unavailable", nil)

	// ErrNotSupported is returned for operations a KMS adapter does not implement.
	ErrNotSupported = errors.New(CodeNotSupported, "kms operation not supported", nil)
)
