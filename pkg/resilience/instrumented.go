package resilience

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedCircuitBreaker wraps a Breaker with logging and tracing.
type InstrumentedCircuitBreaker struct {
	next   Breaker
	name   string
	tracer trace.Tracer
}

// NewInstrumentedCircuitBreaker wraps an existing Breaker with observability.
func NewInstrumentedCircuitBreaker(next Breaker, name string) *InstrumentedCircuitBreaker {
	if name == "" {
		name = "circuit-breaker"
	}
	return &InstrumentedCircuitBreaker{
		next:   next,
		name:   name,
		tracer: otel.Tracer("pkg/resilience"),
	}
}

// NewInstrumentedBreakerFromConfig creates a concrete circuit breaker and wraps
// it with instrumentation. Prefer this when callers want logging without
// changing NewCircuitBreaker's return type.
func NewInstrumentedBreakerFromConfig(cfg CircuitBreakerConfig) *InstrumentedCircuitBreaker {
	name := cfg.Name
	userHook := cfg.OnStateChange
	cfg.OnStateChange = func(n string, from, to State) {
		logger.L().Info("circuit breaker state change",
			"name", n,
			"from", string(from),
			"to", string(to),
		)
		if userHook != nil {
			userHook(n, from, to)
		}
	}
	return NewInstrumentedCircuitBreaker(NewCircuitBreaker(cfg), name)
}

// Execute runs fn under the wrapped breaker with a trace span.
func (i *InstrumentedCircuitBreaker) Execute(ctx context.Context, fn Executor) error {
	ctx, span := i.tracer.Start(ctx, "resilience.CircuitBreaker.Execute",
		trace.WithAttributes(attribute.String("circuitbreaker.name", i.name)),
	)
	defer span.End()

	err := i.next.Execute(ctx, fn)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if errors.Is(err, ErrCircuitOpen) {
			logger.L().WarnContext(ctx, "circuit breaker rejected request",
				"name", i.name,
				"state", string(i.next.State()),
			)
		}
	}
	span.SetAttributes(attribute.String("circuitbreaker.state", string(i.next.State())))
	return err
}

// State returns the wrapped breaker state.
func (i *InstrumentedCircuitBreaker) State() State {
	return i.next.State()
}

// Reset resets the wrapped breaker.
func (i *InstrumentedCircuitBreaker) Reset() {
	logger.L().Info("circuit breaker reset", "name", i.name)
	i.next.Reset()
}

// Metrics returns wrapped breaker metrics.
func (i *InstrumentedCircuitBreaker) Metrics() CircuitBreakerMetrics {
	return i.next.Metrics()
}

// ForceOpen forces the wrapped breaker open when supported.
func (i *InstrumentedCircuitBreaker) ForceOpen() {
	if f, ok := i.next.(interface{ ForceOpen() }); ok {
		logger.L().Warn("circuit breaker force open", "name", i.name)
		f.ForceOpen()
	}
}

// ForceClose forces the wrapped breaker closed when supported.
func (i *InstrumentedCircuitBreaker) ForceClose() {
	if f, ok := i.next.(interface{ ForceClose() }); ok {
		logger.L().Info("circuit breaker force close", "name", i.name)
		f.ForceClose()
	}
}

var _ Breaker = (*InstrumentedCircuitBreaker)(nil)
