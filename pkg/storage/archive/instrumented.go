package archive

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

// InstrumentedStore wraps an ArchiveStore with logging and tracing.
type InstrumentedStore struct {
	next   ArchiveStore
	name   string
	tracer trace.Tracer
}

// NewInstrumentedStore creates a new instrumented archive store wrapper.
func NewInstrumentedStore(store ArchiveStore, name string) *InstrumentedStore {
	return &InstrumentedStore{
		next:   store,
		name:   name,
		tracer: otel.Tracer("pkg/storage/archive"),
	}
}

func (s *InstrumentedStore) startSpan(ctx context.Context, op string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := s.tracer.Start(ctx, fmt.Sprintf("%s.%s", s.name, op))
	span.SetAttributes(attrs...)
	return ctx, span
}

func (s *InstrumentedStore) Archive(ctx context.Context, key string, data io.Reader, opts ArchiveOptions) error {
	ctx, span := s.startSpan(ctx, "Archive",
		attribute.String("archive.key", key),
		attribute.String("archive.class", string(opts.StorageClass)),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "archiving object", "key", key, "class", opts.StorageClass)

	start := time.Now()
	err := s.next.Archive(ctx, key, data, opts)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to archive object", "key", key, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "archived object", "key", key, "duration", duration)
	return nil
}

func (s *InstrumentedStore) Restore(ctx context.Context, key string, opts RestoreOptions) (*RestoreJob, error) {
	ctx, span := s.startSpan(ctx, "Restore",
		attribute.String("archive.key", key),
		attribute.String("archive.tier", string(opts.Tier)),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "initiating restore", "key", key, "tier", opts.Tier)

	job, err := s.next.Restore(ctx, key, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to initiate restore", "key", key, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("restore.job_id", job.ID))
	logger.L().InfoContext(ctx, "restore initiated", "key", key, "job_id", job.ID)
	return job, nil
}

func (s *InstrumentedStore) GetRestoreStatus(ctx context.Context, key string) (*RestoreJob, error) {
	ctx, span := s.startSpan(ctx, "GetRestoreStatus", attribute.String("archive.key", key))
	defer span.End()

	logger.L().DebugContext(ctx, "getting restore status", "key", key)

	job, err := s.next.GetRestoreStatus(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get restore status", "key", key, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("restore.status", string(job.Status)))
	return job, nil
}

func (s *InstrumentedStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	ctx, span := s.startSpan(ctx, "Download", attribute.String("archive.key", key))
	defer span.End()

	logger.L().DebugContext(ctx, "downloading restored object", "key", key)

	rc, err := s.next.Download(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to download object", "key", key, "error", err)
		return nil, err
	}

	return rc, nil
}

func (s *InstrumentedStore) Delete(ctx context.Context, key string) error {
	ctx, span := s.startSpan(ctx, "Delete", attribute.String("archive.key", key))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting archived object", "key", key)

	err := s.next.Delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete object", "key", key, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted archived object", "key", key)
	return nil
}

func (s *InstrumentedStore) GetObject(ctx context.Context, key string) (*ArchiveObject, error) {
	ctx, span := s.startSpan(ctx, "GetObject", attribute.String("archive.key", key))
	defer span.End()

	logger.L().DebugContext(ctx, "getting archive object metadata", "key", key)

	obj, err := s.next.GetObject(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get object metadata", "key", key, "error", err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("archive.size", obj.Size),
		attribute.String("archive.class", string(obj.StorageClass)),
	)
	return obj, nil
}

func (s *InstrumentedStore) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	ctx, span := s.startSpan(ctx, "List",
		attribute.String("archive.prefix", opts.Prefix),
		attribute.Int("archive.limit", opts.Limit),
	)
	defer span.End()

	logger.L().DebugContext(ctx, "listing archived objects", "prefix", opts.Prefix)

	result, err := s.next.List(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to list objects", "prefix", opts.Prefix, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("archive.count", len(result.Objects)))
	return result, nil
}
