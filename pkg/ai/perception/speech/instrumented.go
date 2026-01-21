package speech

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedSpeechClient wraps a SpeechClient with telemetry.
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

func (c *InstrumentedSpeechClient) SpeechToText(ctx context.Context, audio []byte) (string, error) {
	ctx, span := c.tracer.Start(ctx, "SpeechClient.SpeechToText",
		trace.WithAttributes(attribute.Int("speech.audio_size", len(audio))),
	)
	defer span.End()

	start := time.Now()
	text, err := c.next.SpeechToText(ctx, audio)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "stt failed",
			"error", err,
			"duration", duration.String(),
		)
		return "", err
	}

	logger.L().InfoContext(ctx, "stt completed",
		"text_len", len(text),
		"duration", duration.String(),
	)

	return text, nil
}

func (c *InstrumentedSpeechClient) TextToSpeech(ctx context.Context, text string, format AudioFormat) ([]byte, error) {
	ctx, span := c.tracer.Start(ctx, "SpeechClient.TextToSpeech",
		trace.WithAttributes(
			attribute.Int("speech.text_len", len(text)),
			attribute.String("speech.format", string(format)),
		),
	)
	defer span.End()

	start := time.Now()
	audio, err := c.next.TextToSpeech(ctx, text, format)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "tts failed",
			"error", err,
			"duration", duration.String(),
		)
		return nil, err
	}

	logger.L().InfoContext(ctx, "tts completed",
		"audio_size", len(audio),
		"duration", duration.String(),
	)

	return audio, nil
}
