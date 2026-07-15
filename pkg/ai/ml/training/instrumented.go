package training

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ Trainer = (*InstrumentedTrainer)(nil)

// InstrumentedTrainer wraps Trainer with logging and tracing.
type InstrumentedTrainer struct {
	next   Trainer
	tracer trace.Tracer
}

// NewInstrumentedTrainer creates an instrumented training backend.
func NewInstrumentedTrainer(next Trainer) *InstrumentedTrainer {
	return &InstrumentedTrainer{
		next:   next,
		tracer: otel.Tracer("pkg/ai/ml/training"),
	}
}

func (t *InstrumentedTrainer) StartJob(ctx context.Context, config JobConfig) (*Job, error) {
	ctx, span := t.tracer.Start(ctx, "training.StartJob", trace.WithAttributes(
		attribute.String("job.name", config.Name),
		attribute.String("job.model", config.Model),
	))
	defer span.End()
	logger.L().InfoContext(ctx, "start training job", "name", config.Name, "model", config.Model)
	job, err := t.next.StartJob(ctx, config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "start training job failed", "name", config.Name, "error", err)
	}
	return job, err
}

func (t *InstrumentedTrainer) GetJob(ctx context.Context, jobID string) (*Job, error) {
	ctx, span := t.tracer.Start(ctx, "training.GetJob", trace.WithAttributes(
		attribute.String("job.id", jobID),
	))
	defer span.End()
	return t.next.GetJob(ctx, jobID)
}

func (t *InstrumentedTrainer) ListJobs(ctx context.Context) ([]*Job, error) {
	ctx, span := t.tracer.Start(ctx, "training.ListJobs")
	defer span.End()
	return t.next.ListJobs(ctx)
}

func (t *InstrumentedTrainer) StopJob(ctx context.Context, jobID string) error {
	ctx, span := t.tracer.Start(ctx, "training.StopJob", trace.WithAttributes(
		attribute.String("job.id", jobID),
	))
	defer span.End()
	err := t.next.StopJob(ctx, jobID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "stop training job failed", "job_id", jobID, "error", err)
	}
	return err
}

func (t *InstrumentedTrainer) GetMetrics(ctx context.Context, jobID string) ([]Metrics, error) {
	ctx, span := t.tracer.Start(ctx, "training.GetMetrics", trace.WithAttributes(
		attribute.String("job.id", jobID),
	))
	defer span.End()
	return t.next.GetMetrics(ctx, jobID)
}

func (t *InstrumentedTrainer) GetLogs(ctx context.Context, jobID string, tail int) ([]string, error) {
	ctx, span := t.tracer.Start(ctx, "training.GetLogs", trace.WithAttributes(
		attribute.String("job.id", jobID),
		attribute.Int("tail", tail),
	))
	defer span.End()
	return t.next.GetLogs(ctx, jobID, tail)
}

func (t *InstrumentedTrainer) ListCheckpoints(ctx context.Context, jobID string) ([]Checkpoint, error) {
	ctx, span := t.tracer.Start(ctx, "training.ListCheckpoints", trace.WithAttributes(
		attribute.String("job.id", jobID),
	))
	defer span.End()
	return t.next.ListCheckpoints(ctx, jobID)
}
