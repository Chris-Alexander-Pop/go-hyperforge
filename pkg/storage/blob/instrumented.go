package blob

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedStore wraps a Store with logging and tracing
type InstrumentedStore struct {
	next Store
	name string
}

// NewInstrumentedStore creates a new decorator
func NewInstrumentedStore(store Store, name string) *InstrumentedStore {
	return &InstrumentedStore{
		next: store,
		name: name,
	}
}

func (s *InstrumentedStore) Upload(ctx context.Context, key string, data io.Reader) error {
	ctx, span := s.startSpan(ctx, "Upload")
	defer span.End()
	span.SetAttributes(attribute.String("blob.key", key))

	logger.L().InfoContext(ctx, "uploading blob", "key", key)

	start := time.Now()
	err := s.next.Upload(ctx, key, data)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to upload blob", "key", key, "error", err, "duration", duration)
		return err
	}

	logger.L().InfoContext(ctx, "uploaded blob", "key", key, "duration", duration)
	return nil
}

func (s *InstrumentedStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	ctx, span := s.startSpan(ctx, "Download")
	defer span.End()
	span.SetAttributes(attribute.String("blob.key", key))

	logger.L().DebugContext(ctx, "downloading blob", "key", key)

	rc, err := s.next.Download(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to download blob", "key", key, "error", err)
		return nil, err
	}

	return rc, nil
}

func (s *InstrumentedStore) Delete(ctx context.Context, key string) error {
	ctx, span := s.startSpan(ctx, "Delete")
	defer span.End()
	span.SetAttributes(attribute.String("blob.key", key))

	logger.L().InfoContext(ctx, "deleting blob", "key", key)

	err := s.next.Delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete blob", "key", key, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted blob", "key", key)
	return nil
}

func (s *InstrumentedStore) URL(key string) string {
	return s.next.URL(key)
}

func (s *InstrumentedStore) startSpan(ctx context.Context, op string) (context.Context, trace.Span) {
	tracer := otel.Tracer("pkg/blob")
	return tracer.Start(ctx, fmt.Sprintf("%s.%s", s.name, op))
}
