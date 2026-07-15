package auth

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Domain error codes for authentication operations.
const (
	CodeInvalidToken     = "AUTH_INVALID_TOKEN"
	CodeInvalidCreds     = "AUTH_INVALID_CREDENTIALS"
	CodeInvalidConfig    = "AUTH_INVALID_CONFIG"
	CodeSessionNotFound  = "AUTH_SESSION_NOT_FOUND"
	CodeUnauthorized     = "AUTH_UNAUTHORIZED"
	CodeExchangeFailed   = "AUTH_EXCHANGE_FAILED"
	CodeClientNotFound   = "AUTH_CLIENT_NOT_FOUND"
	CodeInvalidGrant     = "AUTH_INVALID_GRANT"
	CodeInvalidClient    = "AUTH_INVALID_CLIENT"
	CodeUnsupportedGrant = "AUTH_UNSUPPORTED_GRANT"
)

// Sentinel errors shared across auth adapters.
var (
	// ErrInvalidToken is returned when a token fails verification.
	ErrInvalidToken = errors.New(CodeInvalidToken, "invalid or expired token", nil)

	// ErrInvalidCredentials is returned when username/password authentication fails.
	ErrInvalidCredentials = errors.New(CodeInvalidCreds, "invalid credentials", nil)

	// ErrInvalidConfig is returned when adapter configuration is incomplete or invalid.
	ErrInvalidConfig = errors.New(CodeInvalidConfig, "invalid auth configuration", nil)

	// ErrSessionNotFound is returned when a session ID does not exist or has expired.
	ErrSessionNotFound = errors.New(CodeSessionNotFound, "session not found", nil)

	// ErrUnauthorized is returned when the caller is not authenticated.
	ErrUnauthorized = errors.New(CodeUnauthorized, "unauthorized", nil)

	// ErrExchangeFailed is returned when an OAuth/OIDC code exchange fails.
	ErrExchangeFailed = errors.New(CodeExchangeFailed, "token exchange failed", nil)

	// ErrClientNotFound is returned when an OAuth2 client_id is unknown.
	ErrClientNotFound = errors.New(CodeClientNotFound, "oauth2 client not found", nil)

	// ErrInvalidGrant is returned for invalid authorization codes or refresh tokens.
	ErrInvalidGrant = errors.New(CodeInvalidGrant, "invalid grant", nil)

	// ErrInvalidClient is returned when client authentication fails.
	ErrInvalidClient = errors.New(CodeInvalidClient, "invalid client", nil)

	// ErrUnsupportedGrant is returned when a grant_type is not enabled.
	ErrUnsupportedGrant = errors.New(CodeUnsupportedGrant, "unsupported grant type", nil)
)

// ErrInvalidTokenWrap wraps an underlying verification error.
func ErrInvalidTokenWrap(err error) *errors.AppError {
	return errors.New(CodeInvalidToken, "invalid or expired token", err)
}

// ErrInvalidConfigMsg creates a configuration error with a detail message.
func ErrInvalidConfigMsg(msg string, err error) *errors.AppError {
	return errors.New(CodeInvalidConfig, "invalid auth configuration: "+msg, err)
}

// IsInvalidToken reports whether err indicates an invalid token.
func IsInvalidToken(err error) bool {
	return errors.Is(err, ErrInvalidToken) || errors.IsCode(err, CodeInvalidToken) || errors.IsCode(err, errors.CodeUnauthorized)
}

// IsInvalidCredentials reports whether err indicates bad credentials.
func IsInvalidCredentials(err error) bool {
	return errors.Is(err, ErrInvalidCredentials) || errors.IsCode(err, CodeInvalidCreds)
}

// IsInvalidConfig reports whether err indicates bad configuration.
func IsInvalidConfig(err error) bool {
	return errors.Is(err, ErrInvalidConfig) || errors.IsCode(err, CodeInvalidConfig)
}
