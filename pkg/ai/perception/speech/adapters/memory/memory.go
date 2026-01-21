package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/perception/speech"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// SpeechClient implements speech.SpeechClient using mock data.
type SpeechClient struct{}

// New creates a new in-memory speech client.
func New() *SpeechClient {
	return &SpeechClient{}
}

func (c *SpeechClient) SpeechToText(ctx context.Context, audio []byte) (string, error) {
	if len(audio) == 0 {
		return "", errors.InvalidArgument("audio content is required", nil)
	}
	return "This is a mock transcription of the audio.", nil
}

func (c *SpeechClient) TextToSpeech(ctx context.Context, text string, format speech.AudioFormat) ([]byte, error) {
	if text == "" {
		return nil, errors.InvalidArgument("text input is required", nil)
	}
	// Return some dummy bytes representing audio
	return []byte("RIFFmockWAVEfmt datamockaudiobytes"), nil
}
