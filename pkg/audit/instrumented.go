package audit

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Ensure compile-time interface compliance.
var (
	_ Auditor = (*InstrumentedAuditor)(nil)
	_ Store   = (*InstrumentedStore)(nil)
)

// InstrumentedAuditor wraps an Auditor with logging and tracing.
type InstrumentedAuditor struct {
	next   Auditor
	tracer trace.Tracer
}

// NewInstrumentedAuditor creates a new instrumented auditor.
func NewInstrumentedAuditor(next Auditor) *InstrumentedAuditor {
	return &InstrumentedAuditor{
		next:   next,
		tracer: otel.Tracer("pkg/audit"),
	}
}

// Log traces and logs an audit write.
func (a *InstrumentedAuditor) Log(ctx context.Context, event Event) error {
	ctx, span := a.tracer.Start(ctx, "audit.Log", trace.WithAttributes(
		attribute.String("event.type", string(event.EventType)),
		attribute.String("event.outcome", string(event.Outcome)),
		attribute.String("actor.id", event.ActorID),
	))
	defer span.End()

	err := a.next.Log(ctx, event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "audit log failed",
			"event_type", string(event.EventType),
			"error", err,
		)
		return err
	}
	return nil
}

// LogWithBuilder returns a builder bound to this instrumented auditor so Send
// goes through the instrumented Log path.
func (a *InstrumentedAuditor) LogWithBuilder(ctx context.Context, eventType EventType) *EventBuilder {
	return newEventBuilder(a, ctx, eventType)
}

// InstrumentedStore wraps a Store with logging and tracing.
type InstrumentedStore struct {
	next   Store
	tracer trace.Tracer
}

// NewInstrumentedStore decorates next with logging and tracing.
func NewInstrumentedStore(next Store) *InstrumentedStore {
	return &InstrumentedStore{
		next:   next,
		tracer: otel.Tracer("pkg/audit"),
	}
}

// Append traces and logs a store append.
func (s *InstrumentedStore) Append(ctx context.Context, event Event) error {
	ctx, span := s.tracer.Start(ctx, "audit.Store.Append", trace.WithAttributes(
		attribute.String("event.type", string(event.EventType)),
		attribute.String("event.outcome", string(event.Outcome)),
	))
	defer span.End()

	err := s.next.Append(ctx, event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "audit store append failed",
			"event_type", string(event.EventType),
			"error", err,
		)
		return err
	}
	return nil
}

// Query traces and logs a store query.
func (s *InstrumentedStore) Query(ctx context.Context, filter QueryFilter) ([]Event, error) {
	ctx, span := s.tracer.Start(ctx, "audit.Store.Query", trace.WithAttributes(
		attribute.String("filter.actor_id", filter.ActorID),
		attribute.String("filter.event_type", string(filter.EventType)),
		attribute.Int("filter.limit", filter.Limit),
	))
	defer span.End()

	events, err := s.next.Query(ctx, filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "audit store query failed", "error", err)
		return nil, err
	}
	span.SetAttributes(attribute.Int("event.count", len(events)))
	return events, nil
}
