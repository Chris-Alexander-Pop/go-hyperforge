package mfa

import (
	"context"
	"time"
)

// Config configures the MFA provider.
type Config struct {
	// Provider specifies the MFA provider implementation.
	// Known values: "memory" (TOTP), "redis" (TOTP), "sms", "email".
	Provider string `env:"AUTH_MFA_PROVIDER" env-default:"memory"`

	// EncryptionKey is used to encrypt MFA secrets (optional).
	EncryptionKey string `env:"AUTH_MFA_ENCRYPTION_KEY"`

	// TOTP configuration (if relevant for the provider).
	TOTPIssuer string `env:"AUTH_MFA_TOTP_ISSUER" env-default:"MyApp"`
	TOTPDigits int    `env:"AUTH_MFA_TOTP_DIGITS" env-default:"6"`
	TOTPPeriod int    `env:"AUTH_MFA_TOTP_PERIOD" env-default:"30"`

	// Channel OTP (SMS/email) settings.
	// CodeDigits is the length of delivered one-time codes (default 6).
	CodeDigits int `env:"AUTH_MFA_CODE_DIGITS" env-default:"6"`
	// CodeTTL is how long a delivered challenge remains valid.
	CodeTTL time.Duration `env:"AUTH_MFA_CODE_TTL" env-default:"5m"`
	// MessageTemplate is used when formatting the OTP body.
	// Must contain a single %s verb for the code (e.g. "Your code is %s").
	MessageTemplate string `env:"AUTH_MFA_MESSAGE_TEMPLATE" env-default:"Your verification code is %s"`
	// EmailSubject is used by the email channel adapter.
	EmailSubject string `env:"AUTH_MFA_EMAIL_SUBJECT" env-default:"Your verification code"`
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

// Provider defines the interface for Multi-Factor Authentication (typically TOTP).
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

// ChannelProvider is MFA delivered via SMS or email.
//
// Adapters accept pkg/communication sms.Sender / email.Sender interfaces.
// Production wiring example (Twilio):
//
//	smsSender, err := twilio.New(sms.Config{...})
//	mfaSMS, err := smscadapter.New(smsSender, mfa.Config{})
//
// For local tests, inject communication/sms/adapters/memory or email/adapters/memory.
type ChannelProvider interface {
	// Enroll registers a destination (E.164 phone or email) and returns recovery codes.
	// A challenge is sent immediately; CompleteEnrollment must succeed before Verify.
	Enroll(ctx context.Context, userID, destination string) (recoveryCodes []string, err error)

	// CompleteEnrollment verifies the enrollment challenge and enables MFA.
	CompleteEnrollment(ctx context.Context, userID, code string) error

	// SendChallenge generates and delivers a new one-time code to the enrolled destination.
	SendChallenge(ctx context.Context, userID string) error

	// Verify validates a challenge code for an enabled enrollment.
	Verify(ctx context.Context, userID, code string) (bool, error)

	// Recover allows login using a recovery code (single-use).
	Recover(ctx context.Context, userID, code string) (bool, error)

	// Disable removes MFA enrollment for a user.
	Disable(ctx context.Context, userID string) error
}
