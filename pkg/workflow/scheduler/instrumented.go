package scheduler

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedScheduler wraps Scheduler with logging and OpenTelemetry spans.
type InstrumentedScheduler struct {
	next   *Scheduler
	tracer trace.Tracer
}

// NewInstrumentedScheduler creates an instrumented scheduler wrapper.
func NewInstrumentedScheduler(next *Scheduler) *InstrumentedScheduler {
	return &InstrumentedScheduler{
		next:   next,
		tracer: otel.Tracer("pkg/workflow/scheduler"),
	}
}

// Underlying returns the wrapped scheduler.
func (i *InstrumentedScheduler) Underlying() *Scheduler {
	return i.next
}

// Schedule registers a cron job with observability.
func (i *InstrumentedScheduler) Schedule(name, schedule string, handler JobFunc) error {
	logger.L().Info("scheduling job", "name", name, "schedule", schedule)
	err := i.next.Schedule(name, schedule, handler)
	if err != nil {
		logger.L().Error("schedule failed", "name", name, "error", err)
	}
	return err
}

// ScheduleOnce registers a one-time job with observability.
func (i *InstrumentedScheduler) ScheduleOnce(name string, runAt time.Time, handler JobFunc) error {
	logger.L().Info("scheduling one-time job", "name", name, "run_at", runAt)
	return i.next.ScheduleOnce(name, runAt, handler)
}

// Start begins the scheduler loop.
func (i *InstrumentedScheduler) Start(ctx context.Context) error {
	logger.L().InfoContext(ctx, "starting scheduler")
	return i.next.Start(ctx)
}

// Stop stops the scheduler.
func (i *InstrumentedScheduler) Stop() {
	logger.L().Info("stopping scheduler")
	i.next.Stop()
}

// SetTickInterval delegates to the underlying scheduler.
func (i *InstrumentedScheduler) SetTickInterval(d time.Duration) {
	i.next.SetTickInterval(d)
}

// GetJob retrieves a job by name.
func (i *InstrumentedScheduler) GetJob(name string) (*Job, error) {
	return i.next.GetJob(name)
}

// ListJobs returns all registered jobs.
func (i *InstrumentedScheduler) ListJobs() []*Job {
	return i.next.ListJobs()
}

// EnableJob enables a job.
func (i *InstrumentedScheduler) EnableJob(name string) error {
	return i.next.EnableJob(name)
}

// DisableJob disables a job.
func (i *InstrumentedScheduler) DisableJob(name string) error {
	return i.next.DisableJob(name)
}

// RunNow immediately executes a job with tracing.
func (i *InstrumentedScheduler) RunNow(ctx context.Context, name string) (*JobExecution, error) {
	ctx, span := i.tracer.Start(ctx, "scheduler.RunNow", trace.WithAttributes(
		attribute.String("job.name", name),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "running job now", "name", name)

	exec, err := i.next.RunNow(ctx, name)
	if exec != nil {
		span.SetAttributes(
			attribute.String("job.execution_id", exec.ID),
			attribute.String("job.status", string(exec.Status)),
		)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "job run failed", "name", name, "error", err)
		return exec, err
	}
	return exec, nil
}
