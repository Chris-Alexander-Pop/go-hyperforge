// Package mfa provides multi-factor authentication utilities.
//
// This package includes:
//   - TOTP: Time-based One-Time Password (Google Authenticator compatible)
//   - Recovery codes: Backup authentication codes
//
// Package otp provides one-time password utilities.
package otp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// TOTPConfig configures TOTP generation.
type TOTPConfig struct {
	// Issuer is your application name (shown in authenticator apps).
	Issuer string

	// Digits is the number of digits in the code (default: 6).
	Digits int

	// Period is the time step in seconds (default: 30).
	Period int

	// Skew is the number of periods to check before/after current (default: 1).
	Skew int

	// SecretLength is the length of generated secrets (default: 20).
	SecretLength int
}

// DefaultTOTPConfig returns sensible defaults.
func DefaultTOTPConfig() TOTPConfig {
	return TOTPConfig{
		Issuer:       "MyApp",
		Digits:       6,
		Period:       30,
		Skew:         1,
		SecretLength: 20,
	}
}

// TOTP handles Time-based One-Time Password operations.
type TOTP struct {
	config TOTPConfig
}

// NewTOTP creates a new TOTP handler.
func NewTOTP(cfg TOTPConfig) *TOTP {
	if cfg.Digits == 0 {
		cfg.Digits = 6
	}
	if cfg.Period == 0 {
		cfg.Period = 30
	}
	if cfg.Skew == 0 {
		cfg.Skew = 1
	}
	if cfg.SecretLength == 0 {
		cfg.SecretLength = 20
	}
	return &TOTP{config: cfg}
}

// GenerateSecret generates a new TOTP secret.
func (t *TOTP) GenerateSecret() (string, error) {
	secret := make([]byte, t.config.SecretLength)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateCode generates a TOTP code for the current time.
func (t *TOTP) GenerateCode(secret string) (string, error) {
	return t.GenerateCodeAt(secret, time.Now())
}

// GenerateCodeAt generates a TOTP code for a specific time.
func (t *TOTP) GenerateCodeAt(secret string, at time.Time) (string, error) {
	counter := uint64(at.Unix()) / uint64(t.config.Period)
	return t.generateCodeForCounter(secret, counter)
}

func (t *TOTP) generateCodeForCounter(secret string, counter uint64) (string, error) {
	// Decode secret
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}

	// Convert counter to bytes
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	// HMAC-SHA1
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0xf
	code := int64(hash[offset]&0x7f)<<24 |
		int64(hash[offset+1]&0xff)<<16 |
		int64(hash[offset+2]&0xff)<<8 |
		int64(hash[offset+3]&0xff)

	// Modulo to get the right number of digits
	mod := int64(1)
	for i := 0; i < t.config.Digits; i++ {
		mod *= 10
	}
	code = code % mod

	// Pad with zeros
	format := fmt.Sprintf("%%0%dd", t.config.Digits)
	return fmt.Sprintf(format, code), nil
}

// Validate checks if a code is valid for the given secret.
func (t *TOTP) Validate(secret, code string) bool {
	valid, _ := t.ValidateReturningCounter(secret, code)
	return valid
}

// ValidateReturningCounter checks if a code is valid and returns the matched counter.
func (t *TOTP) ValidateReturningCounter(secret, code string) (bool, uint64) {
	return t.ValidateAtReturningCounter(secret, code, time.Now())
}

// ValidateAt checks if a code is valid for a specific time.
func (t *TOTP) ValidateAt(secret, code string, at time.Time) bool {
	valid, _ := t.ValidateAtReturningCounter(secret, code, at)
	return valid
}

// ValidateAtReturningCounter checks if a code is valid for a specific time and returns the matched counter.
func (t *TOTP) ValidateAtReturningCounter(secret, code string, at time.Time) (bool, uint64) {
	counter := uint64(at.Unix()) / uint64(t.config.Period)

	// Check current and adjacent time steps
	for i := -t.config.Skew; i <= t.config.Skew; i++ {
		usedCounter := counter + uint64(i)
		expected, err := t.generateCodeForCounter(secret, usedCounter)
		if err != nil {
			continue
		}
		if subtle(expected, code) {
			return true, usedCounter
		}
	}
	return false, 0
}

// ProvisioningURI returns the URI for QR code generation.
// This URI can be encoded as a QR code and scanned by authenticator apps.
func (t *TOTP) ProvisioningURI(secret, accountName string) string {
	return fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		t.config.Issuer,
		accountName,
		secret,
		t.config.Issuer,
		t.config.Digits,
		t.config.Period,
	)
}

// subtle performs constant-time comparison to prevent timing attacks.
func subtle(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
