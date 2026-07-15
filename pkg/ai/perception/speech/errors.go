package speech

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel / helpers for speech cloud adapters.
var (
	// ErrEmptyAudio is returned when STT is called with no audio.
	ErrEmptyAudio = errors.InvalidArgument("audio content is required", nil)

	// ErrEmptyText is returned when TTS is called with empty text.
	ErrEmptyText = errors.InvalidArgument("text input is required", nil)

	// ErrProviderUnavailable is returned when the speech backend cannot be reached.
	ErrProviderUnavailable = errors.Unavailable("speech provider unavailable", nil)
)

// MapHTTPStatus maps an HTTP status from Polly/Transcribe/Google into a domain error.
func MapHTTPStatus(statusCode int, body string) error {
	msg := body
	if msg == "" {
		msg = "speech provider error"
	}
	return errors.FromHTTP(statusCode, msg)
}
