package captcha

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
)

// Config configures the captcha system.
type Config struct {
	// Provider selects the captcha backend.
	// Implemented: "memory", "recaptcha" (HTTP siteverify skeleton).
	// Reserved: "hcaptcha", "turnstile".
	Provider string `env:"SECURITY_CAPTCHA_PROVIDER" env-default:"memory" validate:"required"`

	// SecretKey is the server-side secret for remote verification (reCAPTCHA).
	SecretKey string `env:"SECURITY_CAPTCHA_SECRET"`

	// SiteKey is the client-side site key (informational; not required for Verify).
	SiteKey string `env:"SECURITY_CAPTCHA_SITE_KEY"`
}

// DefaultConfig returns Config with package defaults.
func DefaultConfig() Config {
	return Config{Provider: security.ProviderMemory}
}

// Validate checks Config with pkg/validator.
func (c Config) Validate() error {
	if c.Provider == "" {
		return errors.New(CodeVerifyFailed, "captcha provider is required", nil)
	}
	if err := validator.New().ValidateStruct(context.Background(), c); err != nil {
		if errors.IsCode(err, errors.CodeInvalidArgument) {
			return err
		}
		return errors.New(CodeVerifyFailed, "invalid captcha config", err)
	}
	if c.Provider == security.ProviderRecaptcha && c.SecretKey == "" {
		return errors.New(CodeInvalidToken, "recaptcha secret key is required", nil)
	}
	return nil
}

// Verifier defines the interface for captcha verification.
// Remote adapters (reCAPTCHA) implement the same Verify contract.
type Verifier interface {
	Verify(ctx context.Context, token string) error
}
