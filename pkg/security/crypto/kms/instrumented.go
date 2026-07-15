package kms

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedKeyManager wraps a KeyManager with telemetry.
type InstrumentedKeyManager struct {
	next   KeyManager
	tracer trace.Tracer
}

// NewInstrumentedKeyManager creates a new InstrumentedKeyManager.
func NewInstrumentedKeyManager(next KeyManager) *InstrumentedKeyManager {
	return &InstrumentedKeyManager{
		next:   next,
		tracer: otel.Tracer("pkg/security/crypto/kms"),
	}
}

func (m *InstrumentedKeyManager) Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	ctx, span := m.tracer.Start(ctx, "KeyManager.Encrypt",
		trace.WithAttributes(attribute.String("kms.key_id", keyID)),
	)
	defer span.End()

	start := time.Now()
	ciphertext, err := m.next.Encrypt(ctx, keyID, plaintext)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "kms encrypt failed", "error", err, "key_id", keyID)
		return nil, err
	}

	logger.L().DebugContext(ctx, "kms encrypted data",
		"key_id", keyID,
		"duration", time.Since(start).String(),
		"plaintext_len", len(plaintext),
		"ciphertext_len", len(ciphertext),
	)
	return ciphertext, nil
}

func (m *InstrumentedKeyManager) Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	ctx, span := m.tracer.Start(ctx, "KeyManager.Decrypt",
		trace.WithAttributes(attribute.String("kms.key_id", keyID)),
	)
	defer span.End()

	start := time.Now()
	plaintext, err := m.next.Decrypt(ctx, keyID, ciphertext)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "kms decrypt failed", "error", err, "key_id", keyID)
		return nil, err
	}

	logger.L().DebugContext(ctx, "kms decrypted data",
		"key_id", keyID,
		"duration", time.Since(start).String(),
	)
	return plaintext, nil
}
