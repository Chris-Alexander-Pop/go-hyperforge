package container

import (
	"context"
	"io"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedContainerRuntime wraps a ContainerRuntime with observability.
type InstrumentedContainerRuntime struct {
	next   ContainerRuntime
	tracer trace.Tracer
}

// NewInstrumentedContainerRuntime creates a new instrumented container runtime.
func NewInstrumentedContainerRuntime(next ContainerRuntime) *InstrumentedContainerRuntime {
	return &InstrumentedContainerRuntime{
		next:   next,
		tracer: otel.Tracer("pkg/compute/container"),
	}
}

func (i *InstrumentedContainerRuntime) Create(ctx context.Context, opts CreateOptions) (*Container, error) {
	ctx, span := i.tracer.Start(ctx, "container.Create", trace.WithAttributes(
		attribute.String("container.name", opts.Name),
		attribute.String("container.image", opts.Image),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating container", "name", opts.Name, "image", opts.Image)

	container, err := i.next.Create(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "container creation failed", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("container.id", container.ID))
	return container, nil
}

func (i *InstrumentedContainerRuntime) Get(ctx context.Context, containerID string) (*Container, error) {
	ctx, span := i.tracer.Start(ctx, "container.Get", trace.WithAttributes(
		attribute.String("container.id", containerID),
	))
	defer span.End()

	container, err := i.next.Get(ctx, containerID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return container, nil
}

func (i *InstrumentedContainerRuntime) List(ctx context.Context, opts ListOptions) ([]*Container, error) {
	ctx, span := i.tracer.Start(ctx, "container.List", trace.WithAttributes(
		attribute.Bool("container.all", opts.All),
		attribute.Int("container.limit", opts.Limit),
	))
	defer span.End()

	containers, err := i.next.List(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("container.count", len(containers)))
	return containers, nil
}

func (i *InstrumentedContainerRuntime) Start(ctx context.Context, containerID string) error {
	ctx, span := i.tracer.Start(ctx, "container.Start", trace.WithAttributes(
		attribute.String("container.id", containerID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "starting container", "id", containerID)

	err := i.next.Start(ctx, containerID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "container start failed", "id", containerID, "error", err)
		return err
	}
	return nil
}

func (i *InstrumentedContainerRuntime) Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	ctx, span := i.tracer.Start(ctx, "container.Stop", trace.WithAttributes(
		attribute.String("container.id", containerID),
		attribute.Int64("container.timeout_ms", timeout.Milliseconds()),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "stopping container", "id", containerID)

	err := i.next.Stop(ctx, containerID, timeout)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "container stop failed", "id", containerID, "error", err)
		return err
	}
	return nil
}

func (i *InstrumentedContainerRuntime) Kill(ctx context.Context, containerID string, signal string) error {
	ctx, span := i.tracer.Start(ctx, "container.Kill", trace.WithAttributes(
		attribute.String("container.id", containerID),
		attribute.String("container.signal", signal),
	))
	defer span.End()

	err := i.next.Kill(ctx, containerID, signal)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (i *InstrumentedContainerRuntime) Remove(ctx context.Context, containerID string, force bool) error {
	ctx, span := i.tracer.Start(ctx, "container.Remove", trace.WithAttributes(
		attribute.String("container.id", containerID),
		attribute.Bool("container.force", force),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "removing container", "id", containerID, "force", force)

	err := i.next.Remove(ctx, containerID, force)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "container remove failed", "id", containerID, "error", err)
		return err
	}
	return nil
}

func (i *InstrumentedContainerRuntime) Logs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error) {
	ctx, span := i.tracer.Start(ctx, "container.Logs", trace.WithAttributes(
		attribute.String("container.id", containerID),
		attribute.Bool("container.follow", follow),
	))
	defer span.End()

	logs, err := i.next.Logs(ctx, containerID, follow)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return logs, nil
}

func (i *InstrumentedContainerRuntime) Exec(ctx context.Context, containerID string, opts ExecOptions) (*ExecResult, error) {
	ctx, span := i.tracer.Start(ctx, "container.Exec", trace.WithAttributes(
		attribute.String("container.id", containerID),
	))
	defer span.End()

	result, err := i.next.Exec(ctx, containerID, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("container.exit_code", result.ExitCode))
	return result, nil
}

func (i *InstrumentedContainerRuntime) Wait(ctx context.Context, containerID string) (int, error) {
	ctx, span := i.tracer.Start(ctx, "container.Wait", trace.WithAttributes(
		attribute.String("container.id", containerID),
	))
	defer span.End()

	exitCode, err := i.next.Wait(ctx, containerID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	span.SetAttributes(attribute.Int("container.exit_code", exitCode))
	return exitCode, nil
}

func (i *InstrumentedContainerRuntime) Stats(ctx context.Context, containerID string) (*ContainerStats, error) {
	ctx, span := i.tracer.Start(ctx, "container.Stats", trace.WithAttributes(
		attribute.String("container.id", containerID),
	))
	defer span.End()

	stats, err := i.next.Stats(ctx, containerID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return stats, nil
}

var _ ContainerRuntime = (*InstrumentedContainerRuntime)(nil)
