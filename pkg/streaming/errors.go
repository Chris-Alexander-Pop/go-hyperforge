package streaming

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Error codes for streaming operations.
const (
	CodePutFailed      = "STREAMING_PUT_FAILED"
	CodeClosed         = "STREAMING_CLOSED"
	CodeInvalidConfig  = "STREAMING_INVALID_CONFIG"
	CodeBufferFull     = "STREAMING_BUFFER_FULL"
	CodeStreamNotFound = "STREAMING_STREAM_NOT_FOUND"
)

// ErrClosed is returned when operating on a closed Client.
var ErrClosed = errors.New(CodeClosed, "streaming client is closed", nil)

// ErrBufferFull is returned when the in-memory buffer has reached Config.BufferSize.
var ErrBufferFull = errors.New(CodeBufferFull, "streaming buffer is full", nil)

// ErrPutFailed creates an error for PutRecord failures.
func ErrPutFailed(streamName string, err error) *errors.AppError {
	return errors.New(CodePutFailed, "failed to put record to stream: "+streamName, err)
}

// ErrInvalidConfig creates an error for invalid configuration.
func ErrInvalidConfig(msg string, err error) *errors.AppError {
	return errors.New(CodeInvalidConfig, "invalid streaming configuration: "+msg, err)
}

// ErrStreamNotFound creates an error for a missing stream.
func ErrStreamNotFound(streamName string, err error) *errors.AppError {
	return errors.New(CodeStreamNotFound, "stream not found: "+streamName, err)
}

// IsClosed reports whether err indicates a closed client.
func IsClosed(err error) bool {
	return errors.Is(err, ErrClosed) || errors.IsCode(err, CodeClosed)
}

// IsBufferFull reports whether err indicates a full buffer.
func IsBufferFull(err error) bool {
	return errors.Is(err, ErrBufferFull) || errors.IsCode(err, CodeBufferFull)
}
