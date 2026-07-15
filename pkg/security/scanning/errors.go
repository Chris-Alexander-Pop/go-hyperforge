package scanning

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

const (
	CodeInvalidResource = "SCAN_INVALID_RESOURCE"
	CodeScanFailed      = "SCAN_FAILED"
	CodeUnavailable     = "SCAN_UNAVAILABLE"
)

var (
	// ErrInvalidResource is returned when the scan target is malformed.
	ErrInvalidResource = errors.New(CodeInvalidResource, "invalid scan resource", nil)

	// ErrScanFailed is returned when a scan cannot be completed.
	ErrScanFailed = errors.New(CodeScanFailed, "security scan failed", nil)

	// ErrUnavailable is returned when a remote scanner is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "scanner unavailable", nil)
)
