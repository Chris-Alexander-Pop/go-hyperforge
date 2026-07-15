package controlplane

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedControlPlane wraps a ControlPlane with logging and tracing.
type InstrumentedControlPlane struct {
	next   ControlPlane
	tracer trace.Tracer
}

// NewInstrumentedControlPlane creates a new instrumented control plane.
func NewInstrumentedControlPlane(next ControlPlane) *InstrumentedControlPlane {
	return &InstrumentedControlPlane{
		next:   next,
		tracer: otel.Tracer("pkg/cloud/controlplane"),
	}
}

func (c *InstrumentedControlPlane) RegisterHost(ctx context.Context, host cloud.Host) error {
	ctx, span := c.tracer.Start(ctx, "controlplane.RegisterHost", trace.WithAttributes(
		attribute.String("host.id", host.ID),
		attribute.String("host.name", host.Name),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "registering host", "id", host.ID, "name", host.Name)

	err := c.next.RegisterHost(ctx, host)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to register host", "id", host.ID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "host registered", "id", host.ID)
	return nil
}

func (c *InstrumentedControlPlane) DeregisterHost(ctx context.Context, hostID string) error {
	ctx, span := c.tracer.Start(ctx, "controlplane.DeregisterHost", trace.WithAttributes(
		attribute.String("host.id", hostID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deregistering host", "id", hostID)

	err := c.next.DeregisterHost(ctx, hostID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to deregister host", "id", hostID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "host deregistered", "id", hostID)
	return nil
}

func (c *InstrumentedControlPlane) UpdateHostStatus(ctx context.Context, hostID string, status cloud.HostStatus) error {
	ctx, span := c.tracer.Start(ctx, "controlplane.UpdateHostStatus", trace.WithAttributes(
		attribute.String("host.id", hostID),
		attribute.String("host.status", string(status)),
	))
	defer span.End()

	err := c.next.UpdateHostStatus(ctx, hostID, status)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func (c *InstrumentedControlPlane) GetHost(ctx context.Context, hostID string) (*cloud.Host, error) {
	ctx, span := c.tracer.Start(ctx, "controlplane.GetHost", trace.WithAttributes(
		attribute.String("host.id", hostID),
	))
	defer span.End()

	host, err := c.next.GetHost(ctx, hostID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return host, nil
}

func (c *InstrumentedControlPlane) ListHosts(ctx context.Context) ([]cloud.Host, error) {
	ctx, span := c.tracer.Start(ctx, "controlplane.ListHosts")
	defer span.End()

	hosts, err := c.next.ListHosts(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("host.count", len(hosts)))
	return hosts, nil
}

func (c *InstrumentedControlPlane) CreateInstance(ctx context.Context, req CreateInstanceRequest) (*Instance, error) {
	ctx, span := c.tracer.Start(ctx, "controlplane.CreateInstance", trace.WithAttributes(
		attribute.String("instance.name", req.Name),
		attribute.String("host.id", req.HostID),
	))
	defer span.End()

	inst, err := c.next.CreateInstance(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.String("instance.id", inst.ID))
	return inst, nil
}

func (c *InstrumentedControlPlane) BindInstance(ctx context.Context, instanceID, hostID string) error {
	ctx, span := c.tracer.Start(ctx, "controlplane.BindInstance", trace.WithAttributes(
		attribute.String("instance.id", instanceID),
		attribute.String("host.id", hostID),
	))
	defer span.End()

	err := c.next.BindInstance(ctx, instanceID, hostID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (c *InstrumentedControlPlane) UnbindInstance(ctx context.Context, instanceID string) error {
	ctx, span := c.tracer.Start(ctx, "controlplane.UnbindInstance", trace.WithAttributes(
		attribute.String("instance.id", instanceID),
	))
	defer span.End()

	err := c.next.UnbindInstance(ctx, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (c *InstrumentedControlPlane) GetInstance(ctx context.Context, instanceID string) (*Instance, error) {
	ctx, span := c.tracer.Start(ctx, "controlplane.GetInstance", trace.WithAttributes(
		attribute.String("instance.id", instanceID),
	))
	defer span.End()

	inst, err := c.next.GetInstance(ctx, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return inst, nil
}

func (c *InstrumentedControlPlane) ListInstances(ctx context.Context, opts ListInstancesOptions) ([]Instance, error) {
	ctx, span := c.tracer.Start(ctx, "controlplane.ListInstances")
	defer span.End()

	list, err := c.next.ListInstances(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.Int("instance.count", len(list)))
	return list, nil
}

func (c *InstrumentedControlPlane) DeleteInstance(ctx context.Context, instanceID string) error {
	ctx, span := c.tracer.Start(ctx, "controlplane.DeleteInstance", trace.WithAttributes(
		attribute.String("instance.id", instanceID),
	))
	defer span.End()

	err := c.next.DeleteInstance(ctx, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (c *InstrumentedControlPlane) UpdateInstanceStatus(ctx context.Context, instanceID string, status cloud.InstanceStatus) error {
	ctx, span := c.tracer.Start(ctx, "controlplane.UpdateInstanceStatus", trace.WithAttributes(
		attribute.String("instance.id", instanceID),
		attribute.String("instance.status", string(status)),
	))
	defer span.End()

	err := c.next.UpdateInstanceStatus(ctx, instanceID, status)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

var _ ControlPlane = (*InstrumentedControlPlane)(nil)
