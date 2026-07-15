package captcha

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

const (
	CodeInvalidToken = "CAPTCHA_INVALID_TOKEN"
	CodeVerifyFailed = "CAPTCHA_VERIFY_FAILED"
	CodeUnavailable  = "CAPTCHA_UNAVAILABLE"
)

var (
	// ErrInvalidToken is returned when the captcha token is missing or rejected.
	ErrInvalidToken = errors.New(CodeInvalidToken, "invalid captcha token", nil)

	// ErrVerifyFailed is returned when the provider cannot verify the token.
	ErrVerifyFailed = errors.New(CodeVerifyFailed, "captcha verification failed", nil)

	// ErrUnavailable is returned when the remote captcha API is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "captcha provider unavailable", nil)
)
