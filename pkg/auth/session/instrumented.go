package session

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedManager wraps a Manager with observability.
type InstrumentedManager struct {
	next   Manager
	tracer trace.Tracer
}

// NewInstrumentedManager creates a new InstrumentedManager.
func NewInstrumentedManager(next Manager) *InstrumentedManager {
	return &InstrumentedManager{
		next:   next,
		tracer: otel.Tracer("pkg/auth/session"),
	}
}

func (m *InstrumentedManager) Create(ctx context.Context, userID string, metadata map[string]interface{}) (*Session, error) {
	ctx, span := m.tracer.Start(ctx, "session.Create", trace.WithAttributes(
		attribute.String("user.id", userID),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "creating session", "user_id", userID)

	s, err := m.next.Create(ctx, userID, metadata)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create session", "error", err, "user_id", userID)
		return nil, err
	}
	span.SetAttributes(attribute.String("session.id", s.ID))
	return s, nil
}

func (m *InstrumentedManager) Get(ctx context.Context, sessionID string) (*Session, error) {
	ctx, span := m.tracer.Start(ctx, "session.Get", trace.WithAttributes(
		attribute.String("session.id", sessionID),
	))
	defer span.End()

	s, err := m.next.Get(ctx, sessionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Lower log level as "session not found" might be common
		logger.L().DebugContext(ctx, "failed to get session", "error", err, "session_id", sessionID)
		return nil, err
	}
	span.SetAttributes(attribute.String("user.id", s.UserID))
	return s, nil
}

func (m *InstrumentedManager) Delete(ctx context.Context, sessionID string) error {
	ctx, span := m.tracer.Start(ctx, "session.Delete", trace.WithAttributes(
		attribute.String("session.id", sessionID),
	))
	defer span.End()

	err := m.next.Delete(ctx, sessionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete session", "error", err, "session_id", sessionID)
		return err
	}
	logger.L().DebugContext(ctx, "session deleted", "session_id", sessionID)
	return nil
}

func (m *InstrumentedManager) Refresh(ctx context.Context, sessionID string) (*Session, error) {
	ctx, span := m.tracer.Start(ctx, "session.Refresh", trace.WithAttributes(
		attribute.String("session.id", sessionID),
	))
	defer span.End()

	s, err := m.next.Refresh(ctx, sessionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to refresh session", "error", err, "session_id", sessionID)
		return nil, err
	}
	return s, nil
}
