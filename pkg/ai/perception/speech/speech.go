package speech

import (
	"context"
)

// Config configures the speech service.
type Config struct {
	// Provider specifies the speech provider (memory, aws-polly, google-speech).
	Provider string `env:"AI_PERCEPTION_SPEECH_PROVIDER" env-default:"memory"`
}

// AudioFormat represents the audio encoding.
type AudioFormat string

const (
	FormatMP3  AudioFormat = "mp3"
	FormatWAV  AudioFormat = "wav"
	FormatOGG  AudioFormat = "ogg"
	FormatFLAC AudioFormat = "flac"
)

// Voice represents a synthetic voice.
type Voice struct {
	ID       string `json:"id"`
	Language string `json:"language"`
	Gender   string `json:"gender"`
	Name     string `json:"name"`
}

// SpeechClient defines the interface for STT and TTS operations.
type SpeechClient interface {
	SpeechToText(ctx context.Context, audio []byte) (string, error)
	TextToSpeech(ctx context.Context, text string, format AudioFormat) ([]byte, error)
}
