package crypto

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedEncryptor wraps an Encryptor with logging and tracing.
// Spans use context.Background because Encryptor methods are context-free.
type InstrumentedEncryptor struct {
	next   Encryptor
	tracer trace.Tracer
}

// Ensure InstrumentedEncryptor implements Encryptor.
var _ Encryptor = (*InstrumentedEncryptor)(nil)

// NewInstrumentedEncryptor creates an observability wrapper around next.
func NewInstrumentedEncryptor(next Encryptor) *InstrumentedEncryptor {
	return &InstrumentedEncryptor{
		next:   next,
		tracer: otel.Tracer("pkg/security/crypto"),
	}
}

// Encrypt encrypts plaintext and records telemetry (never logs plaintext).
func (e *InstrumentedEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	ctx, span := e.tracer.Start(context.Background(), "Encryptor.Encrypt")
	defer span.End()

	start := time.Now()
	out, err := e.next.Encrypt(plaintext)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "encrypt failed", "error", err, "duration", time.Since(start).String())
		return nil, err
	}

	logger.L().DebugContext(ctx, "encrypt ok", "plaintext_bytes", len(plaintext), "duration", time.Since(start).String())
	return out, nil
}

// Decrypt decrypts ciphertext and records telemetry (never logs plaintext).
func (e *InstrumentedEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	ctx, span := e.tracer.Start(context.Background(), "Encryptor.Decrypt")
	defer span.End()

	start := time.Now()
	out, err := e.next.Decrypt(ciphertext)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().WarnContext(ctx, "decrypt failed", "error", err, "duration", time.Since(start).String())
		return nil, err
	}

	logger.L().DebugContext(ctx, "decrypt ok", "ciphertext_bytes", len(ciphertext), "duration", time.Since(start).String())
	return out, nil
}
