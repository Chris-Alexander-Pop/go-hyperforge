package provisioning

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedProvisioner wraps a Provisioner with logging and tracing.
type InstrumentedProvisioner struct {
	next   Provisioner
	tracer trace.Tracer
}

// NewInstrumentedProvisioner creates a new instrumented provisioner.
func NewInstrumentedProvisioner(next Provisioner) *InstrumentedProvisioner {
	return &InstrumentedProvisioner{
		next:   next,
		tracer: otel.Tracer("pkg/cloud/provisioning"),
	}
}

func (p *InstrumentedProvisioner) ProvisionHost(ctx context.Context, hostID string, imageURL string) error {
	ctx, span := p.tracer.Start(ctx, "provisioning.ProvisionHost", trace.WithAttributes(
		attribute.String("host.id", hostID),
		attribute.String("image.url", imageURL),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "provisioning host", "id", hostID, "image", imageURL)

	err := p.next.ProvisionHost(ctx, hostID, imageURL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to provision host", "id", hostID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "host provisioned", "id", hostID)
	return nil
}

func (p *InstrumentedProvisioner) DeprovisionHost(ctx context.Context, hostID string) error {
	ctx, span := p.tracer.Start(ctx, "provisioning.DeprovisionHost", trace.WithAttributes(
		attribute.String("host.id", hostID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deprovisioning host", "id", hostID)

	err := p.next.DeprovisionHost(ctx, hostID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to deprovision host", "id", hostID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "host deprovisioned", "id", hostID)
	return nil
}

func (p *InstrumentedProvisioner) GetHostStatus(ctx context.Context, hostID string) (cloud.HostStatus, error) {
	ctx, span := p.tracer.Start(ctx, "provisioning.GetHostStatus", trace.WithAttributes(
		attribute.String("host.id", hostID),
	))
	defer span.End()

	status, err := p.next.GetHostStatus(ctx, hostID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return cloud.HostStatusUnknown, err
	}

	span.SetAttributes(attribute.String("host.status", string(status)))
	return status, nil
}

func (p *InstrumentedProvisioner) PowerCycle(ctx context.Context, hostID string) error {
	ctx, span := p.tracer.Start(ctx, "provisioning.PowerCycle", trace.WithAttributes(
		attribute.String("host.id", hostID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "power cycling host", "id", hostID)

	err := p.next.PowerCycle(ctx, hostID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to power cycle host", "id", hostID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "host power cycled", "id", hostID)
	return nil
}
