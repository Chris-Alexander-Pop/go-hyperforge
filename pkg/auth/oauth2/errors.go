package oauth2

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Re-export auth domain codes for oauth2 callers that import this package only.
const (
	CodeClientNotFound   = "AUTH_CLIENT_NOT_FOUND"
	CodeInvalidGrant     = "AUTH_INVALID_GRANT"
	CodeInvalidClient    = "AUTH_INVALID_CLIENT"
	CodeUnsupportedGrant = "AUTH_UNSUPPORTED_GRANT"
	CodeInvalidRequest   = "AUTH_INVALID_REQUEST"
)

var (
	ErrClientNotFound   = errors.New(CodeClientNotFound, "oauth2 client not found", nil)
	ErrInvalidGrant     = errors.New(CodeInvalidGrant, "invalid grant", nil)
	ErrInvalidClient    = errors.New(CodeInvalidClient, "invalid client", nil)
	ErrUnsupportedGrant = errors.New(CodeUnsupportedGrant, "unsupported grant type", nil)
	ErrInvalidRequest   = errors.New(CodeInvalidRequest, "invalid oauth2 request", nil)
)

// ErrInvalidRequestMsg creates a detailed invalid-request error.
func ErrInvalidRequestMsg(msg string) *errors.AppError {
	return errors.New(CodeInvalidRequest, msg, nil)
}
