package apigateway

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedManager wraps an APIGatewayManager with logging and tracing.
type InstrumentedManager struct {
	next   APIGatewayManager
	tracer trace.Tracer
}

// NewInstrumentedManager creates a new instrumented API gateway manager.
func NewInstrumentedManager(next APIGatewayManager) *InstrumentedManager {
	return &InstrumentedManager{
		next:   next,
		tracer: otel.Tracer("pkg/network/apigateway"),
	}
}

func (m *InstrumentedManager) CreateAPI(ctx context.Context, opts CreateAPIOptions) (*API, error) {
	ctx, span := m.tracer.Start(ctx, "apigateway.CreateAPI", trace.WithAttributes(
		attribute.String("api.name", opts.Name),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating API", "name", opts.Name)

	api, err := m.next.CreateAPI(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create API", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("api.id", api.ID))
	logger.L().InfoContext(ctx, "API created", "id", api.ID, "name", opts.Name)
	return api, nil
}

func (m *InstrumentedManager) GetAPI(ctx context.Context, id string) (*API, error) {
	ctx, span := m.tracer.Start(ctx, "apigateway.GetAPI", trace.WithAttributes(
		attribute.String("api.id", id),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "getting API", "id", id)

	api, err := m.next.GetAPI(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get API", "id", id, "error", err)
		return nil, err
	}
	return api, nil
}

func (m *InstrumentedManager) ListAPIs(ctx context.Context) ([]*API, error) {
	ctx, span := m.tracer.Start(ctx, "apigateway.ListAPIs")
	defer span.End()

	logger.L().DebugContext(ctx, "listing APIs")

	apis, err := m.next.ListAPIs(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to list APIs", "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("api.count", len(apis)))
	return apis, nil
}

func (m *InstrumentedManager) DeleteAPI(ctx context.Context, id string) error {
	ctx, span := m.tracer.Start(ctx, "apigateway.DeleteAPI", trace.WithAttributes(
		attribute.String("api.id", id),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting API", "id", id)

	err := m.next.DeleteAPI(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete API", "id", id, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "API deleted", "id", id)
	return nil
}

func (m *InstrumentedManager) AddRoute(ctx context.Context, apiID string, route Route) (*Route, error) {
	ctx, span := m.tracer.Start(ctx, "apigateway.AddRoute", trace.WithAttributes(
		attribute.String("api.id", apiID),
		attribute.String("route.method", route.Method),
		attribute.String("route.path", route.Path),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "adding route", "api_id", apiID, "method", route.Method, "path", route.Path)

	r, err := m.next.AddRoute(ctx, apiID, route)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to add route", "api_id", apiID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("route.id", r.ID))
	logger.L().InfoContext(ctx, "route added", "route_id", r.ID)
	return r, nil
}

func (m *InstrumentedManager) RemoveRoute(ctx context.Context, apiID, routeID string) error {
	ctx, span := m.tracer.Start(ctx, "apigateway.RemoveRoute", trace.WithAttributes(
		attribute.String("api.id", apiID),
		attribute.String("route.id", routeID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "removing route", "api_id", apiID, "route_id", routeID)

	err := m.next.RemoveRoute(ctx, apiID, routeID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to remove route", "route_id", routeID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "route removed", "route_id", routeID)
	return nil
}

func (m *InstrumentedManager) Deploy(ctx context.Context, apiID, stageName string) (*Stage, error) {
	ctx, span := m.tracer.Start(ctx, "apigateway.Deploy", trace.WithAttributes(
		attribute.String("api.id", apiID),
		attribute.String("stage.name", stageName),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deploying API", "api_id", apiID, "stage", stageName)

	stage, err := m.next.Deploy(ctx, apiID, stageName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to deploy API", "api_id", apiID, "error", err)
		return nil, err
	}

	logger.L().InfoContext(ctx, "API deployed", "api_id", apiID, "stage", stageName)
	return stage, nil
}

func (m *InstrumentedManager) GetStage(ctx context.Context, apiID, stageName string) (*Stage, error) {
	ctx, span := m.tracer.Start(ctx, "apigateway.GetStage", trace.WithAttributes(
		attribute.String("api.id", apiID),
		attribute.String("stage.name", stageName),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "getting stage", "api_id", apiID, "stage", stageName)

	stage, err := m.next.GetStage(ctx, apiID, stageName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get stage", "api_id", apiID, "stage", stageName, "error", err)
		return nil, err
	}
	return stage, nil
}
