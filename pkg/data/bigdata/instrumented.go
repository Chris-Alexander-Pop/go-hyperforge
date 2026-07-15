package bigdata

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedClient wraps a Client with tracing and logging.
type InstrumentedClient struct {
	next   Client
	name   string
	tracer trace.Tracer
}

// NewInstrumentedClient creates a new instrumented bigdata client.
func NewInstrumentedClient(next Client, name string) *InstrumentedClient {
	if name == "" {
		name = "bigdata"
	}
	return &InstrumentedClient{
		next:   next,
		name:   name,
		tracer: otel.Tracer("pkg/data/bigdata"),
	}
}

func (c *InstrumentedClient) Query(ctx context.Context, query string, args ...interface{}) (*Result, error) {
	ctx, span := c.tracer.Start(ctx, c.name+".Query", trace.WithAttributes(
		attribute.String("db.system", "bigdata"),
		attribute.String("db.statement", query),
		attribute.Int("db.args_count", len(args)),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "executing bigdata query",
		"adapter", c.name,
		"query", query,
		"args_count", len(args),
	)

	start := time.Now()
	res, err := c.next.Query(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "bigdata query failed",
			"adapter", c.name,
			"query", query,
			"error", err,
			"duration", duration,
		)
		return nil, err
	}

	rows := 0
	if res != nil {
		rows = len(res.Rows)
	}
	span.SetAttributes(attribute.Int("db.rows_returned", rows))
	logger.L().DebugContext(ctx, "bigdata query complete",
		"adapter", c.name,
		"rows", rows,
		"duration", duration,
	)

	return res, nil
}

func (c *InstrumentedClient) Close() error {
	logger.L().Info("closing bigdata client", "adapter", c.name)
	err := c.next.Close()
	if err != nil {
		logger.L().Error("failed to close bigdata client", "adapter", c.name, "error", err)
	}
	return err
}
