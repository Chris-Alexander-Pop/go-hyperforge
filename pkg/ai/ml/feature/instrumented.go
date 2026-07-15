package feature

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ FeatureStore = (*InstrumentedStore)(nil)

// InstrumentedStore wraps FeatureStore with logging and tracing.
type InstrumentedStore struct {
	next   FeatureStore
	tracer trace.Tracer
}

// NewInstrumentedStore creates an instrumented feature store.
func NewInstrumentedStore(next FeatureStore) *InstrumentedStore {
	return &InstrumentedStore{
		next:   next,
		tracer: otel.Tracer("pkg/ai/ml/feature"),
	}
}

func (s *InstrumentedStore) CreateFeatureGroup(ctx context.Context, group *FeatureGroup) error {
	name := ""
	if group != nil {
		name = group.Name
	}
	ctx, span := s.tracer.Start(ctx, "feature.CreateFeatureGroup", trace.WithAttributes(
		attribute.String("feature.group", name),
	))
	defer span.End()
	logger.L().InfoContext(ctx, "create feature group", "group", name)
	err := s.next.CreateFeatureGroup(ctx, group)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "create feature group failed", "group", name, "error", err)
	}
	return err
}

func (s *InstrumentedStore) GetFeatureGroup(ctx context.Context, name string) (*FeatureGroup, error) {
	ctx, span := s.tracer.Start(ctx, "feature.GetFeatureGroup", trace.WithAttributes(
		attribute.String("feature.group", name),
	))
	defer span.End()
	return s.next.GetFeatureGroup(ctx, name)
}

func (s *InstrumentedStore) ListFeatureGroups(ctx context.Context) ([]*FeatureGroup, error) {
	ctx, span := s.tracer.Start(ctx, "feature.ListFeatureGroups")
	defer span.End()
	return s.next.ListFeatureGroups(ctx)
}

func (s *InstrumentedStore) DeleteFeatureGroup(ctx context.Context, name string) error {
	ctx, span := s.tracer.Start(ctx, "feature.DeleteFeatureGroup", trace.WithAttributes(
		attribute.String("feature.group", name),
	))
	defer span.End()
	err := s.next.DeleteFeatureGroup(ctx, name)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (s *InstrumentedStore) IngestFeatures(ctx context.Context, groupName string, vectors []FeatureVector) error {
	ctx, span := s.tracer.Start(ctx, "feature.IngestFeatures", trace.WithAttributes(
		attribute.String("feature.group", groupName),
		attribute.Int("feature.vectors", len(vectors)),
	))
	defer span.End()
	logger.L().InfoContext(ctx, "ingest features", "group", groupName, "count", len(vectors))
	err := s.next.IngestFeatures(ctx, groupName, vectors)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "ingest features failed", "group", groupName, "error", err)
	}
	return err
}

func (s *InstrumentedStore) GetOnlineFeatures(ctx context.Context, groupName string, entityKeys []string, featureNames []string) ([]FeatureVector, error) {
	ctx, span := s.tracer.Start(ctx, "feature.GetOnlineFeatures", trace.WithAttributes(
		attribute.String("feature.group", groupName),
		attribute.Int("feature.entities", len(entityKeys)),
	))
	defer span.End()
	return s.next.GetOnlineFeatures(ctx, groupName, entityKeys, featureNames)
}

func (s *InstrumentedStore) GetHistoricalFeatures(ctx context.Context, groupName string, entityKeys []string, startTime, endTime time.Time) ([]FeatureVector, error) {
	ctx, span := s.tracer.Start(ctx, "feature.GetHistoricalFeatures", trace.WithAttributes(
		attribute.String("feature.group", groupName),
	))
	defer span.End()
	return s.next.GetHistoricalFeatures(ctx, groupName, entityKeys, startTime, endTime)
}
