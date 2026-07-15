package secrets

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedSecretManager wraps a SecretManager with telemetry.
type InstrumentedSecretManager struct {
	next   SecretManager
	tracer trace.Tracer
}

// Ensure InstrumentedSecretManager implements SecretManager.
var _ SecretManager = (*InstrumentedSecretManager)(nil)

// NewInstrumentedSecretManager creates a new InstrumentedSecretManager.
func NewInstrumentedSecretManager(next SecretManager) *InstrumentedSecretManager {
	return &InstrumentedSecretManager{
		next:   next,
		tracer: otel.Tracer("pkg/security/secrets"),
	}
}

func (m *InstrumentedSecretManager) Get(ctx context.Context, name string) (string, error) {
	ctx, span := m.tracer.Start(ctx, "SecretManager.Get",
		trace.WithAttributes(attribute.String("secret.name", name)),
	)
	defer span.End()

	start := time.Now()
	val, err := m.next.Get(ctx, name)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "secret retrieval failed", "error", err, "name", name)
		return "", err
	}

	logger.L().DebugContext(ctx, "secret retrieved", "name", name, "duration", time.Since(start).String())
	return val, nil
}

func (m *InstrumentedSecretManager) Set(ctx context.Context, name, value string) error {
	ctx, span := m.tracer.Start(ctx, "SecretManager.Set",
		trace.WithAttributes(attribute.String("secret.name", name)),
	)
	defer span.End()

	if err := m.next.Set(ctx, name, value); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "secret set failed", "error", err, "name", name)
		return err
	}

	logger.L().InfoContext(ctx, "secret set", "name", name)
	return nil
}

func (m *InstrumentedSecretManager) Rotate(ctx context.Context, name, newValue string) (string, error) {
	ctx, span := m.tracer.Start(ctx, "SecretManager.Rotate",
		trace.WithAttributes(attribute.String("secret.name", name)),
	)
	defer span.End()

	val, err := m.next.Rotate(ctx, name, newValue)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "secret rotate failed", "error", err, "name", name)
		return "", err
	}

	logger.L().InfoContext(ctx, "secret rotated", "name", name)
	return val, nil
}
