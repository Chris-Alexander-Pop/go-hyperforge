package mfa

import (
	"context"
	"time"
)

// Config configures the MFA provider.
type Config struct {
	// Provider specifies the MFA provider implementation.
	Provider string `env:"AUTH_MFA_PROVIDER" env-default:"memory"`

	// EncryptionKey is used to encrypt MFA secrets (optional).
	EncryptionKey string `env:"AUTH_MFA_ENCRYPTION_KEY"`

	// TOTP configuration (if relevant for the provider).
	TOTPIssuer string `env:"AUTH_MFA_TOTP_ISSUER" env-default:"MyApp"`
	TOTPDigits int    `env:"AUTH_MFA_TOTP_DIGITS" env-default:"6"`
	TOTPPeriod int    `env:"AUTH_MFA_TOTP_PERIOD" env-default:"30"`
}

// Enrollment represents a user's MFA enrollment.
type Enrollment struct {
	UserID     string                 `json:"user_id"`
	Type       string                 `json:"type"` // "totp", "sms", etc.
	Secret     string                 `json:"-"`    // Encrypted secret
	Enabled    bool                   `json:"enabled"`
	Recovery   []string               `json:"-"` // Recovery codes (hashed)
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	LastUsedAt time.Time              `json:"last_used_at,omitempty"`
}

// VerificationResult contains the result of an MFA verification.
type VerificationResult struct {
	Success bool
	Message string
}

// Provider defines the interface for Multi-Factor Authentication.
type Provider interface {
	// Enroll initiates enrollment for a user.
	// Returns a secret (for QR code) and recovery codes.
	Enroll(ctx context.Context, userID string) (secret string, recoveryCodes []string, err error)

	// CompleteEnrollment verifies the code and finalizes enrollment.
	CompleteEnrollment(ctx context.Context, userID, code string) error

	// Verify validates a code for a user.
	Verify(ctx context.Context, userID, code string) (bool, error)

	// Recover allows login using a recovery code.
	Recover(ctx context.Context, userID, code string) (bool, error)

	// Disable disables MFA for a user.
	Disable(ctx context.Context, userID string) error
}
