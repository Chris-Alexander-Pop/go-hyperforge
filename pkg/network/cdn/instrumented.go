package cdn

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedManager wraps a CDNManager with logging and tracing.
type InstrumentedManager struct {
	next   CDNManager
	tracer trace.Tracer
}

// NewInstrumentedManager creates a new instrumented CDN manager.
func NewInstrumentedManager(next CDNManager) *InstrumentedManager {
	return &InstrumentedManager{
		next:   next,
		tracer: otel.Tracer("pkg/network/cdn"),
	}
}

func (m *InstrumentedManager) CreateDistribution(ctx context.Context, opts CreateDistributionOptions) (*Distribution, error) {
	ctx, span := m.tracer.Start(ctx, "cdn.CreateDistribution", trace.WithAttributes(
		attribute.String("cdn.origin", opts.OriginDomain),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating CDN distribution", "origin", opts.OriginDomain)

	dist, err := m.next.CreateDistribution(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create distribution", "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("cdn.distribution_id", dist.ID))
	logger.L().InfoContext(ctx, "CDN distribution created", "id", dist.ID)
	return dist, nil
}

func (m *InstrumentedManager) GetDistribution(ctx context.Context, id string) (*Distribution, error) {
	ctx, span := m.tracer.Start(ctx, "cdn.GetDistribution", trace.WithAttributes(
		attribute.String("cdn.distribution_id", id),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "getting CDN distribution", "id", id)

	dist, err := m.next.GetDistribution(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get distribution", "id", id, "error", err)
		return nil, err
	}
	return dist, nil
}

func (m *InstrumentedManager) ListDistributions(ctx context.Context) ([]*Distribution, error) {
	ctx, span := m.tracer.Start(ctx, "cdn.ListDistributions")
	defer span.End()

	logger.L().DebugContext(ctx, "listing CDN distributions")

	dists, err := m.next.ListDistributions(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to list distributions", "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("cdn.distribution_count", len(dists)))
	return dists, nil
}

func (m *InstrumentedManager) UpdateDistribution(ctx context.Context, id string, opts CreateDistributionOptions) (*Distribution, error) {
	ctx, span := m.tracer.Start(ctx, "cdn.UpdateDistribution", trace.WithAttributes(
		attribute.String("cdn.distribution_id", id),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "updating CDN distribution", "id", id)

	dist, err := m.next.UpdateDistribution(ctx, id, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to update distribution", "id", id, "error", err)
		return nil, err
	}

	logger.L().InfoContext(ctx, "CDN distribution updated", "id", id)
	return dist, nil
}

func (m *InstrumentedManager) DeleteDistribution(ctx context.Context, id string) error {
	ctx, span := m.tracer.Start(ctx, "cdn.DeleteDistribution", trace.WithAttributes(
		attribute.String("cdn.distribution_id", id),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting CDN distribution", "id", id)

	err := m.next.DeleteDistribution(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete distribution", "id", id, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "CDN distribution deleted", "id", id)
	return nil
}

func (m *InstrumentedManager) DisableDistribution(ctx context.Context, id string) error {
	ctx, span := m.tracer.Start(ctx, "cdn.DisableDistribution", trace.WithAttributes(
		attribute.String("cdn.distribution_id", id),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "disabling CDN distribution", "id", id)

	err := m.next.DisableDistribution(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to disable distribution", "id", id, "error", err)
		return err
	}
	return nil
}

func (m *InstrumentedManager) EnableDistribution(ctx context.Context, id string) error {
	ctx, span := m.tracer.Start(ctx, "cdn.EnableDistribution", trace.WithAttributes(
		attribute.String("cdn.distribution_id", id),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "enabling CDN distribution", "id", id)

	err := m.next.EnableDistribution(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to enable distribution", "id", id, "error", err)
		return err
	}
	return nil
}

func (m *InstrumentedManager) Invalidate(ctx context.Context, distributionID string, paths []string) (*Invalidation, error) {
	ctx, span := m.tracer.Start(ctx, "cdn.Invalidate", trace.WithAttributes(
		attribute.String("cdn.distribution_id", distributionID),
		attribute.Int("cdn.path_count", len(paths)),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating CDN invalidation", "distribution_id", distributionID, "paths", len(paths))

	inv, err := m.next.Invalidate(ctx, distributionID, paths)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create invalidation", "distribution_id", distributionID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("cdn.invalidation_id", inv.ID))
	logger.L().InfoContext(ctx, "CDN invalidation created", "id", inv.ID)
	return inv, nil
}

func (m *InstrumentedManager) GetInvalidation(ctx context.Context, distributionID, invalidationID string) (*Invalidation, error) {
	ctx, span := m.tracer.Start(ctx, "cdn.GetInvalidation", trace.WithAttributes(
		attribute.String("cdn.distribution_id", distributionID),
		attribute.String("cdn.invalidation_id", invalidationID),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "getting CDN invalidation", "distribution_id", distributionID, "invalidation_id", invalidationID)

	inv, err := m.next.GetInvalidation(ctx, distributionID, invalidationID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get invalidation", "invalidation_id", invalidationID, "error", err)
		return nil, err
	}
	return inv, nil
}
