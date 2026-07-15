package otp

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// ChannelCodeConfig configures one-time codes delivered via SMS/email.
type ChannelCodeConfig struct {
	Digits int
	TTL    time.Duration
}

// DefaultChannelCodeConfig returns sensible defaults (6 digits, 5 minutes).
func DefaultChannelCodeConfig() ChannelCodeConfig {
	return ChannelCodeConfig{
		Digits: 6,
		TTL:    5 * time.Minute,
	}
}

// GenerateChannelCode returns a numeric OTP of the configured length.
func GenerateChannelCode(cfg ChannelCodeConfig) (string, error) {
	if cfg.Digits <= 0 {
		cfg.Digits = 6
	}
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(cfg.Digits)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", cfg.Digits, n), nil
}

// HashChannelCode normalizes and hashes a channel OTP for at-rest storage.
func HashChannelCode(code string) string {
	normalized := strings.TrimSpace(code)
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", sum)
}
