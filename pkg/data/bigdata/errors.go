package bigdata

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Error codes for bigdata operations.
const (
	CodeQueryFailed     = "BIGDATA_QUERY_FAILED"
	CodeInvalidQuery    = "BIGDATA_INVALID_QUERY"
	CodeConnectionFailed = "BIGDATA_CONNECTION_FAILED"
	CodeClosed          = "BIGDATA_CLOSED"
	CodeTimeout         = "BIGDATA_TIMEOUT"
	CodeNotFound        = "BIGDATA_NOT_FOUND"
)

// ErrClosed is returned when operating on a closed Client.
var ErrClosed = errors.New(CodeClosed, "bigdata client is closed", nil)

// ErrQueryFailed creates an error for query execution failures.
func ErrQueryFailed(query string, err error) *errors.AppError {
	return errors.New(CodeQueryFailed, "bigdata query failed: "+query, err)
}

// ErrInvalidQuery creates an error for malformed queries.
func ErrInvalidQuery(msg string, err error) *errors.AppError {
	return errors.New(CodeInvalidQuery, "invalid bigdata query: "+msg, err)
}

// ErrConnectionFailed creates an error when the backend is unreachable.
func ErrConnectionFailed(backend string, err error) *errors.AppError {
	return errors.New(CodeConnectionFailed, "bigdata backend connection failed: "+backend, err)
}

// ErrTimeout creates an error when a query times out.
func ErrTimeout(err error) *errors.AppError {
	return errors.New(CodeTimeout, "bigdata operation timed out", err)
}

// ErrNotFound creates an error for a missing resource (table, dataset, etc.).
func ErrNotFound(resource string, err error) *errors.AppError {
	return errors.New(CodeNotFound, "bigdata resource not found: "+resource, err)
}

// IsClosed reports whether err indicates a closed client.
func IsClosed(err error) bool {
	return errors.Is(err, ErrClosed) || errors.IsCode(err, CodeClosed)
}

// IsQueryFailed reports whether err indicates a query failure.
func IsQueryFailed(err error) bool {
	return errors.IsCode(err, CodeQueryFailed)
}
