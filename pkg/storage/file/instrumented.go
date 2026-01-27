package file

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

// InstrumentedStore wraps a FileStore with logging and tracing.
type InstrumentedStore struct {
	next   FileStore
	name   string
	tracer trace.Tracer
}

// NewInstrumentedStore creates a new instrumented file store wrapper.
func NewInstrumentedStore(store FileStore, name string) *InstrumentedStore {
	return &InstrumentedStore{
		next:   store,
		name:   name,
		tracer: otel.Tracer("pkg/storage/file"),
	}
}

func (s *InstrumentedStore) startSpan(ctx context.Context, op string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := s.tracer.Start(ctx, fmt.Sprintf("%s.%s", s.name, op))
	span.SetAttributes(attrs...)
	return ctx, span
}

func (s *InstrumentedStore) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	ctx, span := s.startSpan(ctx, "Read", attribute.String("file.path", path))
	defer span.End()

	logger.L().DebugContext(ctx, "reading file", "path", path)

	rc, err := s.next.Read(ctx, path)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to read file", "path", path, "error", err)
		return nil, err
	}

	return rc, nil
}

func (s *InstrumentedStore) Write(ctx context.Context, path string, data io.Reader) error {
	ctx, span := s.startSpan(ctx, "Write", attribute.String("file.path", path))
	defer span.End()

	logger.L().InfoContext(ctx, "writing file", "path", path)

	start := time.Now()
	err := s.next.Write(ctx, path, data)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to write file", "path", path, "error", err, "duration", duration)
		return err
	}

	logger.L().InfoContext(ctx, "wrote file", "path", path, "duration", duration)
	return nil
}

func (s *InstrumentedStore) Delete(ctx context.Context, path string) error {
	ctx, span := s.startSpan(ctx, "Delete", attribute.String("file.path", path))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting file", "path", path)

	err := s.next.Delete(ctx, path)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete file", "path", path, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted file", "path", path)
	return nil
}

func (s *InstrumentedStore) List(ctx context.Context, prefix string, opts ListOptions) ([]FileInfo, error) {
	ctx, span := s.startSpan(ctx, "List",
		attribute.String("file.prefix", prefix),
		attribute.Bool("file.recursive", opts.Recursive),
	)
	defer span.End()

	logger.L().DebugContext(ctx, "listing files", "prefix", prefix, "recursive", opts.Recursive)

	files, err := s.next.List(ctx, prefix, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to list files", "prefix", prefix, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("file.count", len(files)))
	return files, nil
}

func (s *InstrumentedStore) Stat(ctx context.Context, path string) (*FileInfo, error) {
	ctx, span := s.startSpan(ctx, "Stat", attribute.String("file.path", path))
	defer span.End()

	logger.L().DebugContext(ctx, "getting file info", "path", path)

	info, err := s.next.Stat(ctx, path)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to stat file", "path", path, "error", err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("file.size", info.Size),
		attribute.Bool("file.is_dir", info.IsDir),
	)
	return info, nil
}

func (s *InstrumentedStore) Mkdir(ctx context.Context, path string) error {
	ctx, span := s.startSpan(ctx, "Mkdir", attribute.String("file.path", path))
	defer span.End()

	logger.L().InfoContext(ctx, "creating directory", "path", path)

	err := s.next.Mkdir(ctx, path)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create directory", "path", path, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "created directory", "path", path)
	return nil
}

func (s *InstrumentedStore) Rename(ctx context.Context, oldPath, newPath string) error {
	ctx, span := s.startSpan(ctx, "Rename",
		attribute.String("file.old_path", oldPath),
		attribute.String("file.new_path", newPath),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "renaming file", "old_path", oldPath, "new_path", newPath)

	err := s.next.Rename(ctx, oldPath, newPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to rename file", "old_path", oldPath, "new_path", newPath, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "renamed file", "old_path", oldPath, "new_path", newPath)
	return nil
}

func (s *InstrumentedStore) Copy(ctx context.Context, srcPath, dstPath string) error {
	ctx, span := s.startSpan(ctx, "Copy",
		attribute.String("file.src_path", srcPath),
		attribute.String("file.dst_path", dstPath),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "copying file", "src_path", srcPath, "dst_path", dstPath)

	start := time.Now()
	err := s.next.Copy(ctx, srcPath, dstPath)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to copy file", "src_path", srcPath, "dst_path", dstPath, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "copied file", "src_path", srcPath, "dst_path", dstPath, "duration", duration)
	return nil
}
