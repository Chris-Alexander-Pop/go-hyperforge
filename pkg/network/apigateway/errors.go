package apigateway

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for API gateway operations.
var (
	// ErrAPINotFound is returned when an API does not exist.
	ErrAPINotFound = errors.NotFound("API not found", nil)

	// ErrRouteNotFound is returned when a route does not exist.
	ErrRouteNotFound = errors.NotFound("route not found", nil)

	// ErrStageNotFound is returned when a stage does not exist.
	ErrStageNotFound = errors.NotFound("stage not found", nil)

	// ErrInvalidAPIName is returned when an API name is empty or invalid.
	ErrInvalidAPIName = errors.InvalidArgument("invalid API name", nil)
)
