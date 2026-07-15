package network

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for root TCP/UDP servers.
var (
	// ErrListenFailed is returned when binding a TCP or UDP listener fails.
	ErrListenFailed = errors.Internal("failed to listen", nil)

	// ErrAcceptFailed is returned when accepting a TCP connection fails.
	ErrAcceptFailed = errors.Internal("failed to accept connection", nil)

	// ErrReadFailed is returned when a UDP read fails.
	ErrReadFailed = errors.Internal("failed to read packet", nil)
)
