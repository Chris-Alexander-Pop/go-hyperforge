package streaming

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedClient wraps a Client with logging and tracing.
type InstrumentedClient struct {
	next   Client
	tracer trace.Tracer
}

// Ensure InstrumentedClient implements Client.
var _ Client = (*InstrumentedClient)(nil)

// NewInstrumentedClient creates a new InstrumentedClient.
func NewInstrumentedClient(next Client) *InstrumentedClient {
	return &InstrumentedClient{
		next:   next,
		tracer: otel.Tracer("pkg/streaming"),
	}
}

func (c *InstrumentedClient) PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error {
	ctx, span := c.tracer.Start(ctx, "streaming.PutRecord", trace.WithAttributes(
		attribute.String("stream.name", streamName),
		attribute.String("partition.key", partitionKey),
		attribute.Int("data.size", len(data)),
	))
	defer span.End()

	err := c.next.PutRecord(ctx, streamName, partitionKey, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to put record",
			"stream", streamName, "partition_key", partitionKey, "error", err)
		return err
	}

	span.SetStatus(codes.Ok, "record put")
	logger.L().DebugContext(ctx, "put record to stream",
		"stream", streamName, "partition_key", partitionKey, "data_size", len(data))
	return nil
}

func (c *InstrumentedClient) PutRecords(ctx context.Context, records []Record) error {
	ctx, span := c.tracer.Start(ctx, "streaming.PutRecords", trace.WithAttributes(
		attribute.Int("record.count", len(records)),
	))
	defer span.End()

	err := c.next.PutRecords(ctx, records)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to put records", "count", len(records), "error", err)
		return err
	}
	span.SetStatus(codes.Ok, "records put")
	return nil
}

func (c *InstrumentedClient) Close() error {
	logger.L().Info("closing streaming client")
	return c.next.Close()
}
