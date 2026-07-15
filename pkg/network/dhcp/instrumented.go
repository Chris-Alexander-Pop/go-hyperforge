package dhcp

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedIPAM wraps an IPAM interface with logging and tracing.
type InstrumentedIPAM struct {
	next   IPAM
	tracer trace.Tracer
}

// NewInstrumentedIPAM creates a new instrumented IPAM.
func NewInstrumentedIPAM(next IPAM) *InstrumentedIPAM {
	return &InstrumentedIPAM{
		next:   next,
		tracer: otel.Tracer("pkg/network/dhcp"),
	}
}

func (i *InstrumentedIPAM) AllocateIP(ctx context.Context, poolID string) (*IPAllocation, error) {
	ctx, span := i.tracer.Start(ctx, "dhcp.AllocateIP", trace.WithAttributes(
		attribute.String("pool.id", poolID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "allocating ip", "pool_id", poolID)

	alloc, err := i.next.AllocateIP(ctx, poolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to allocate ip", "pool_id", poolID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("ip.address", alloc.IP), attribute.String("alloc.id", alloc.ID))
	logger.L().InfoContext(ctx, "ip allocated", "ip", alloc.IP, "id", alloc.ID)
	return alloc, nil
}

func (i *InstrumentedIPAM) ReserveIP(ctx context.Context, poolID string, ip string) error {
	ctx, span := i.tracer.Start(ctx, "dhcp.ReserveIP", trace.WithAttributes(
		attribute.String("pool.id", poolID),
		attribute.String("ip.address", ip),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "reserving ip", "pool_id", poolID, "ip", ip)

	err := i.next.ReserveIP(ctx, poolID, ip)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to reserve ip", "pool_id", poolID, "ip", ip, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "ip reserved", "ip", ip)
	return nil
}

func (i *InstrumentedIPAM) ReleaseIP(ctx context.Context, allocationID string) error {
	ctx, span := i.tracer.Start(ctx, "dhcp.ReleaseIP", trace.WithAttributes(
		attribute.String("alloc.id", allocationID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "releasing ip", "id", allocationID)

	err := i.next.ReleaseIP(ctx, allocationID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to release ip", "id", allocationID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "ip released", "id", allocationID)
	return nil
}
