package waf

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedManager wraps a Manager with telemetry.
type InstrumentedManager struct {
	next   Manager
	tracer trace.Tracer
}

// NewInstrumentedManager creates a new InstrumentedManager.
func NewInstrumentedManager(next Manager) *InstrumentedManager {
	return &InstrumentedManager{
		next:   next,
		tracer: otel.Tracer("pkg/security/waf"),
	}
}

func (m *InstrumentedManager) BlockIP(ctx context.Context, ip, reason string) error {
	ctx, span := m.tracer.Start(ctx, "Manager.BlockIP",
		trace.WithAttributes(
			attribute.String("waf.ip", ip),
			attribute.String("waf.reason", reason),
		),
	)
	defer span.End()

	if err := m.next.BlockIP(ctx, ip, reason); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "waf block ip failed", "error", err, "ip", ip)
		return err
	}

	logger.L().InfoContext(ctx, "waf blocked ip", "ip", ip, "reason", reason)
	return nil
}

func (m *InstrumentedManager) AllowIP(ctx context.Context, ip string) error {
	ctx, span := m.tracer.Start(ctx, "Manager.AllowIP",
		trace.WithAttributes(attribute.String("waf.ip", ip)),
	)
	defer span.End()

	if err := m.next.AllowIP(ctx, ip); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "waf allow ip failed", "error", err, "ip", ip)
		return err
	}

	logger.L().InfoContext(ctx, "waf allowed ip", "ip", ip)
	return nil
}

func (m *InstrumentedManager) GetRules(ctx context.Context) ([]Rule, error) {
	ctx, span := m.tracer.Start(ctx, "Manager.GetRules")
	defer span.End()

	start := time.Now()
	rules, err := m.next.GetRules(ctx)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	logger.L().DebugContext(ctx, "waf retrieved rules", "count", len(rules), "duration", time.Since(start).String())
	return rules, nil
}
