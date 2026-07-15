package cqrs

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedCommandBus wraps a CommandBus with logging and OpenTelemetry spans.
type InstrumentedCommandBus struct {
	next   *CommandBus
	tracer trace.Tracer
}

// NewInstrumentedCommandBus decorates bus with logging and tracing.
func NewInstrumentedCommandBus(bus *CommandBus) *InstrumentedCommandBus {
	return &InstrumentedCommandBus{
		next:   bus,
		tracer: otel.Tracer("pkg/enterprise/cqrs"),
	}
}

// Register registers a handler on the underlying bus.
func (b *InstrumentedCommandBus) Register(commandName string, handler CommandHandler) {
	b.next.Register(commandName, handler)
}

// RegisterCommand registers a handler using the command's name.
func (b *InstrumentedCommandBus) RegisterCommand(cmd Command, handler CommandHandler) {
	b.next.RegisterCommand(cmd, handler)
}

// Dispatch logs and traces command dispatch.
func (b *InstrumentedCommandBus) Dispatch(ctx context.Context, cmd Command) error {
	name := ""
	if cmd != nil {
		name = cmd.CommandName()
	}
	ctx, span := b.tracer.Start(ctx, "cqrs.CommandBus.Dispatch", trace.WithAttributes(
		attribute.String("command.name", name),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "dispatching command", "command", name)

	err := b.next.Dispatch(ctx, cmd)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "command dispatch failed", "command", name, "error", err)
		return err
	}
	return nil
}

// InstrumentedQueryBus wraps a QueryBus with logging and OpenTelemetry spans.
type InstrumentedQueryBus struct {
	next   *QueryBus
	tracer trace.Tracer
}

// NewInstrumentedQueryBus decorates bus with logging and tracing.
func NewInstrumentedQueryBus(bus *QueryBus) *InstrumentedQueryBus {
	return &InstrumentedQueryBus{
		next:   bus,
		tracer: otel.Tracer("pkg/enterprise/cqrs"),
	}
}

// Register registers a handler on the underlying bus.
func (b *InstrumentedQueryBus) Register(queryName string, handler QueryHandler) {
	b.next.Register(queryName, handler)
}

// RegisterQuery registers a handler using the query's name.
func (b *InstrumentedQueryBus) RegisterQuery(query Query, handler QueryHandler) {
	b.next.RegisterQuery(query, handler)
}

// Dispatch logs and traces query dispatch.
func (b *InstrumentedQueryBus) Dispatch(ctx context.Context, query Query) (interface{}, error) {
	name := ""
	if query != nil {
		name = query.QueryName()
	}
	ctx, span := b.tracer.Start(ctx, "cqrs.QueryBus.Dispatch", trace.WithAttributes(
		attribute.String("query.name", name),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "dispatching query", "query", name)

	result, err := b.next.Dispatch(ctx, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "query dispatch failed", "query", name, "error", err)
		return nil, err
	}
	return result, nil
}
