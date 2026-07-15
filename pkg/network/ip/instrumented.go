package ip

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedIntelligence wraps an IPIntelligence with logging and tracing.
type InstrumentedIntelligence struct {
	next   IPIntelligence
	tracer trace.Tracer
}

// NewInstrumentedIntelligence creates a new instrumented IP intelligence service.
func NewInstrumentedIntelligence(next IPIntelligence) *InstrumentedIntelligence {
	return &InstrumentedIntelligence{
		next:   next,
		tracer: otel.Tracer("pkg/network/ip"),
	}
}

func (i *InstrumentedIntelligence) Lookup(ctx context.Context, ipAddr string) (*GeoLocation, error) {
	ctx, span := i.tracer.Start(ctx, "ip.Lookup", trace.WithAttributes(
		attribute.String("ip.address", ipAddr),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "looking up IP geolocation", "ip", ipAddr)

	loc, err := i.next.Lookup(ctx, ipAddr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IP lookup failed", "ip", ipAddr, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("ip.country", loc.Country))
	return loc, nil
}

func (i *InstrumentedIntelligence) LookupBatch(ctx context.Context, ips []string) ([]*GeoLocation, error) {
	ctx, span := i.tracer.Start(ctx, "ip.LookupBatch", trace.WithAttributes(
		attribute.Int("ip.count", len(ips)),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "batch IP lookup", "count", len(ips))

	locs, err := i.next.LookupBatch(ctx, ips)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "batch IP lookup failed", "error", err)
		return nil, err
	}
	return locs, nil
}

func (i *InstrumentedIntelligence) GetThreatInfo(ctx context.Context, ipAddr string) (*ThreatInfo, error) {
	ctx, span := i.tracer.Start(ctx, "ip.GetThreatInfo", trace.WithAttributes(
		attribute.String("ip.address", ipAddr),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "getting IP threat info", "ip", ipAddr)

	info, err := i.next.GetThreatInfo(ctx, ipAddr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "threat lookup failed", "ip", ipAddr, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Bool("ip.is_threat", info.IsThreat))
	return info, nil
}

func (i *InstrumentedIntelligence) IsBlocked(ctx context.Context, ipAddr string) (bool, error) {
	ctx, span := i.tracer.Start(ctx, "ip.IsBlocked", trace.WithAttributes(
		attribute.String("ip.address", ipAddr),
	))
	defer span.End()

	blocked, err := i.next.IsBlocked(ctx, ipAddr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IsBlocked failed", "ip", ipAddr, "error", err)
		return false, err
	}

	span.SetAttributes(attribute.Bool("ip.blocked", blocked))
	return blocked, nil
}

func (i *InstrumentedIntelligence) IsCountryAllowed(ctx context.Context, ipAddr string, allowedCountries []string) (bool, error) {
	ctx, span := i.tracer.Start(ctx, "ip.IsCountryAllowed", trace.WithAttributes(
		attribute.String("ip.address", ipAddr),
		attribute.Int("ip.allowed_country_count", len(allowedCountries)),
	))
	defer span.End()

	allowed, err := i.next.IsCountryAllowed(ctx, ipAddr, allowedCountries)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IsCountryAllowed failed", "ip", ipAddr, "error", err)
		return false, err
	}

	span.SetAttributes(attribute.Bool("ip.country_allowed", allowed))
	return allowed, nil
}
