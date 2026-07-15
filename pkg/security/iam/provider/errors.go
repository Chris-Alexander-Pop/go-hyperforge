package provider

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrInvalidCredentials is returned when authentication fails.
	ErrInvalidCredentials = errors.Unauthorized("invalid credentials", nil)

	// ErrTokenExpired is returned when a token is no longer valid.
	ErrTokenExpired = errors.Unauthorized("token expired", nil)

	// ErrUserNotFound is returned when a user does not exist.
	ErrUserNotFound = errors.NotFound("user not found", nil)

	// ErrUserAlreadyExists is returned when attempting to create a user that already exists.
	ErrUserAlreadyExists = errors.Conflict("user already exists", nil)
)
