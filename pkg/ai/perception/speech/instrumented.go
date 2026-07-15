package speech

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedSpeechClient wraps SpeechClient with observability.
type InstrumentedSpeechClient struct {
	next   SpeechClient
	tracer trace.Tracer
}

// NewInstrumentedSpeechClient creates a new InstrumentedSpeechClient.
func NewInstrumentedSpeechClient(next SpeechClient) *InstrumentedSpeechClient {
	return &InstrumentedSpeechClient{
		next:   next,
		tracer: otel.Tracer("pkg/ai/perception/speech"),
	}
}

// SpeechToText instruments SpeechToText.
func (c *InstrumentedSpeechClient) SpeechToText(ctx context.Context, audio []byte) (string, error) {
	ctx, span := c.tracer.Start(ctx, "speech.SpeechToText", trace.WithAttributes(
		attribute.Int("audio.size", len(audio)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "speech to text", "audio_size", len(audio))

	text, err := c.next.SpeechToText(ctx, audio)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "speech to text failed", "error", err)
	}

	return text, err
}

// TextToSpeech instruments TextToSpeech.
func (c *InstrumentedSpeechClient) TextToSpeech(ctx context.Context, text string, format AudioFormat) ([]byte, error) {
	ctx, span := c.tracer.Start(ctx, "speech.TextToSpeech", trace.WithAttributes(
		attribute.Int("text.length", len(text)),
		attribute.String("format", string(format)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "text to speech", "text_length", len(text), "format", format)

	audio, err := c.next.TextToSpeech(ctx, text, format)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "text to speech failed", "error", err)
	}

	return audio, err
}
