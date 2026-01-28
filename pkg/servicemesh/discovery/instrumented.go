package discovery

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedServiceRegistry wraps a ServiceRegistry with observability.
type InstrumentedServiceRegistry struct {
	next   ServiceRegistry
	tracer trace.Tracer
}

// NewInstrumentedServiceRegistry creates a new instrumented service registry.
func NewInstrumentedServiceRegistry(next ServiceRegistry) *InstrumentedServiceRegistry {
	return &InstrumentedServiceRegistry{
		next:   next,
		tracer: otel.Tracer("pkg/servicemesh/discovery"),
	}
}

func (i *InstrumentedServiceRegistry) Register(ctx context.Context, opts RegisterOptions) (*Service, error) {
	ctx, span := i.tracer.Start(ctx, "discovery.Register", trace.WithAttributes(
		attribute.String("service.name", opts.Name),
		attribute.String("service.address", opts.Address),
		attribute.Int("service.port", opts.Port),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "registering service", "name", opts.Name, "address", opts.Address)

	svc, err := i.next.Register(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "service registration failed", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("service.id", svc.ID))
	return svc, nil
}

func (i *InstrumentedServiceRegistry) Deregister(ctx context.Context, serviceID string) error {
	ctx, span := i.tracer.Start(ctx, "discovery.Deregister", trace.WithAttributes(
		attribute.String("service.id", serviceID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deregistering service", "id", serviceID)

	err := i.next.Deregister(ctx, serviceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "service deregistration failed", "id", serviceID, "error", err)
		return err
	}
	return nil
}

func (i *InstrumentedServiceRegistry) Lookup(ctx context.Context, serviceName string, opts QueryOptions) ([]*Service, error) {
	ctx, span := i.tracer.Start(ctx, "discovery.Lookup", trace.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.Bool("service.healthy_only", opts.HealthyOnly),
	))
	defer span.End()

	services, err := i.next.Lookup(ctx, serviceName, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("service.count", len(services)))
	return services, nil
}

func (i *InstrumentedServiceRegistry) Get(ctx context.Context, serviceID string) (*Service, error) {
	ctx, span := i.tracer.Start(ctx, "discovery.Get", trace.WithAttributes(
		attribute.String("service.id", serviceID),
	))
	defer span.End()

	svc, err := i.next.Get(ctx, serviceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return svc, nil
}

func (i *InstrumentedServiceRegistry) List(ctx context.Context, opts QueryOptions) ([]*Service, error) {
	ctx, span := i.tracer.Start(ctx, "discovery.List", trace.WithAttributes(
		attribute.String("service.namespace", opts.Namespace),
		attribute.Bool("service.healthy_only", opts.HealthyOnly),
	))
	defer span.End()

	services, err := i.next.List(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("service.count", len(services)))
	return services, nil
}

func (i *InstrumentedServiceRegistry) Watch(ctx context.Context, serviceName string) (<-chan []*Service, error) {
	ctx, span := i.tracer.Start(ctx, "discovery.Watch", trace.WithAttributes(
		attribute.String("service.name", serviceName),
	))
	defer span.End()

	ch, err := i.next.Watch(ctx, serviceName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return ch, nil
}

func (i *InstrumentedServiceRegistry) Heartbeat(ctx context.Context, serviceID string) error {
	ctx, span := i.tracer.Start(ctx, "discovery.Heartbeat", trace.WithAttributes(
		attribute.String("service.id", serviceID),
	))
	defer span.End()

	err := i.next.Heartbeat(ctx, serviceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (i *InstrumentedServiceRegistry) UpdateHealth(ctx context.Context, serviceID string, status HealthStatus) error {
	ctx, span := i.tracer.Start(ctx, "discovery.UpdateHealth", trace.WithAttributes(
		attribute.String("service.id", serviceID),
		attribute.String("service.health_status", string(status)),
	))
	defer span.End()

	err := i.next.UpdateHealth(ctx, serviceID, status)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (i *InstrumentedServiceRegistry) Close() error {
	return i.next.Close()
}

var _ ServiceRegistry = (*InstrumentedServiceRegistry)(nil)
