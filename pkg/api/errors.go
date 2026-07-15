package api

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Domain sentinel errors for the API package.
// HTTP/gRPC status mapping is handled by pkg/errors.HTTPStatus / GRPCStatus.
var (
	ErrUnauthorized    = errors.Unauthorized("unauthorized", nil)
	ErrForbidden       = errors.Forbidden("forbidden", nil)
	ErrNotFound        = errors.NotFound("not found", nil)
	ErrInvalidArgument = errors.InvalidArgument("invalid argument", nil)
	ErrRateLimited     = errors.ResourceExhausted("rate limit exceeded", nil)
	ErrUnavailable     = errors.Unavailable("service unavailable", nil)
)
