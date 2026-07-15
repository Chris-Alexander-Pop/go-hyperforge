package inference

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ InferenceServer = (*InstrumentedServer)(nil)

// InstrumentedServer wraps InferenceServer with logging and tracing.
type InstrumentedServer struct {
	next   InferenceServer
	tracer trace.Tracer
}

// NewInstrumentedServer creates an instrumented inference server.
func NewInstrumentedServer(next InferenceServer) *InstrumentedServer {
	return &InstrumentedServer{
		next:   next,
		tracer: otel.Tracer("pkg/ai/ml/inference"),
	}
}

func (s *InstrumentedServer) LoadModel(ctx context.Context, config Config) (*Model, error) {
	ctx, span := s.tracer.Start(ctx, "inference.LoadModel", trace.WithAttributes(
		attribute.String("model.name", config.Name),
		attribute.String("model.type", string(config.ModelType)),
	))
	defer span.End()
	logger.L().InfoContext(ctx, "loading model", "name", config.Name, "type", config.ModelType)
	model, err := s.next.LoadModel(ctx, config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "load model failed", "name", config.Name, "error", err)
	}
	return model, err
}

func (s *InstrumentedServer) UnloadModel(ctx context.Context, name string) error {
	ctx, span := s.tracer.Start(ctx, "inference.UnloadModel", trace.WithAttributes(
		attribute.String("model.name", name),
	))
	defer span.End()
	err := s.next.UnloadModel(ctx, name)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "unload model failed", "name", name, "error", err)
	}
	return err
}

func (s *InstrumentedServer) GetModel(ctx context.Context, name string) (*Model, error) {
	ctx, span := s.tracer.Start(ctx, "inference.GetModel", trace.WithAttributes(
		attribute.String("model.name", name),
	))
	defer span.End()
	return s.next.GetModel(ctx, name)
}

func (s *InstrumentedServer) ListModels(ctx context.Context) ([]*Model, error) {
	ctx, span := s.tracer.Start(ctx, "inference.ListModels")
	defer span.End()
	return s.next.ListModels(ctx)
}

func (s *InstrumentedServer) Predict(ctx context.Context, request *PredictRequest) (*PredictResponse, error) {
	name := ""
	if request != nil {
		name = request.ModelName
	}
	ctx, span := s.tracer.Start(ctx, "inference.Predict", trace.WithAttributes(
		attribute.String("model.name", name),
	))
	defer span.End()
	logger.L().InfoContext(ctx, "predict", "model", name)
	resp, err := s.next.Predict(ctx, request)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "predict failed", "model", name, "error", err)
	}
	return resp, err
}

func (s *InstrumentedServer) PredictBatch(ctx context.Context, requests []*PredictRequest) ([]*PredictResponse, error) {
	ctx, span := s.tracer.Start(ctx, "inference.PredictBatch", trace.WithAttributes(
		attribute.Int("batch.size", len(requests)),
	))
	defer span.End()
	resp, err := s.next.PredictBatch(ctx, requests)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "predict batch failed", "error", err)
	}
	return resp, err
}

func (s *InstrumentedServer) Health(ctx context.Context) (*HealthStatus, error) {
	ctx, span := s.tracer.Start(ctx, "inference.Health")
	defer span.End()
	return s.next.Health(ctx)
}
