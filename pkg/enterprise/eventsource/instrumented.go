package eventsource

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Ensure InstrumentedEventStore implements EventStore at compile time.
var _ EventStore = (*InstrumentedEventStore)(nil)

// InstrumentedEventStore wraps an EventStore with logging and OpenTelemetry spans.
type InstrumentedEventStore struct {
	next   EventStore
	tracer trace.Tracer
}

// NewInstrumentedEventStore decorates next with logging and tracing.
func NewInstrumentedEventStore(next EventStore) *InstrumentedEventStore {
	return &InstrumentedEventStore{
		next:   next,
		tracer: otel.Tracer("pkg/enterprise/eventsource"),
	}
}

// Append logs and traces an append operation.
func (s *InstrumentedEventStore) Append(ctx context.Context, aggregateID string, expectedVersion int, events []Event) error {
	ctx, span := s.tracer.Start(ctx, "eventsource.Append", trace.WithAttributes(
		attribute.String("aggregate.id", aggregateID),
		attribute.Int("aggregate.expected_version", expectedVersion),
		attribute.Int("event.count", len(events)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "appending events",
		"aggregate_id", aggregateID,
		"expected_version", expectedVersion,
		"count", len(events),
	)

	err := s.next.Append(ctx, aggregateID, expectedVersion, events)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "append failed", "aggregate_id", aggregateID, "error", err)
		return err
	}
	return nil
}

// Load logs and traces a load operation.
func (s *InstrumentedEventStore) Load(ctx context.Context, aggregateID string) ([]Event, error) {
	ctx, span := s.tracer.Start(ctx, "eventsource.Load", trace.WithAttributes(
		attribute.String("aggregate.id", aggregateID),
	))
	defer span.End()

	events, err := s.next.Load(ctx, aggregateID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "load failed", "aggregate_id", aggregateID, "error", err)
		return nil, err
	}
	span.SetAttributes(attribute.Int("event.count", len(events)))
	return events, nil
}

// LoadFrom logs and traces a LoadFrom operation.
func (s *InstrumentedEventStore) LoadFrom(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error) {
	ctx, span := s.tracer.Start(ctx, "eventsource.LoadFrom", trace.WithAttributes(
		attribute.String("aggregate.id", aggregateID),
		attribute.Int("event.from_version", fromVersion),
	))
	defer span.End()

	events, err := s.next.LoadFrom(ctx, aggregateID, fromVersion)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "load_from failed", "aggregate_id", aggregateID, "error", err)
		return nil, err
	}
	span.SetAttributes(attribute.Int("event.count", len(events)))
	return events, nil
}

// LoadAll logs and traces a LoadAll operation.
func (s *InstrumentedEventStore) LoadAll(ctx context.Context) ([]Event, error) {
	ctx, span := s.tracer.Start(ctx, "eventsource.LoadAll")
	defer span.End()

	events, err := s.next.LoadAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "load_all failed", "error", err)
		return nil, err
	}
	span.SetAttributes(attribute.Int("event.count", len(events)))
	return events, nil
}
