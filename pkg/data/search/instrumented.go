package search

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedEngine wraps a SearchEngine with logging and tracing.
type InstrumentedEngine struct {
	next   SearchEngine
	name   string
	tracer trace.Tracer
}

// NewInstrumentedEngine creates a new instrumented search engine wrapper.
func NewInstrumentedEngine(engine SearchEngine, name string) *InstrumentedEngine {
	return &InstrumentedEngine{
		next:   engine,
		name:   name,
		tracer: otel.Tracer("pkg/data/search"),
	}
}

func (s *InstrumentedEngine) startSpan(ctx context.Context, op string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := s.tracer.Start(ctx, fmt.Sprintf("%s.%s", s.name, op))
	span.SetAttributes(attrs...)
	return ctx, span
}

func (s *InstrumentedEngine) CreateIndex(ctx context.Context, indexName string, mapping *IndexMapping) error {
	ctx, span := s.startSpan(ctx, "CreateIndex", attribute.String("search.index", indexName))
	defer span.End()

	logger.L().InfoContext(ctx, "creating index", "index", indexName)

	err := s.next.CreateIndex(ctx, indexName, mapping)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create index", "index", indexName, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "created index", "index", indexName)
	return nil
}

func (s *InstrumentedEngine) DeleteIndex(ctx context.Context, indexName string) error {
	ctx, span := s.startSpan(ctx, "DeleteIndex", attribute.String("search.index", indexName))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting index", "index", indexName)

	err := s.next.DeleteIndex(ctx, indexName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete index", "index", indexName, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted index", "index", indexName)
	return nil
}

func (s *InstrumentedEngine) GetIndex(ctx context.Context, indexName string) (*IndexInfo, error) {
	ctx, span := s.startSpan(ctx, "GetIndex", attribute.String("search.index", indexName))
	defer span.End()

	logger.L().DebugContext(ctx, "getting index info", "index", indexName)

	info, err := s.next.GetIndex(ctx, indexName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get index", "index", indexName, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int64("search.doc_count", info.DocCount))
	return info, nil
}

func (s *InstrumentedEngine) Index(ctx context.Context, indexName, docID string, doc interface{}) error {
	ctx, span := s.startSpan(ctx, "Index",
		attribute.String("search.index", indexName),
		attribute.String("search.doc_id", docID),
	)
	defer span.End()

	logger.L().DebugContext(ctx, "indexing document", "index", indexName, "doc_id", docID)

	err := s.next.Index(ctx, indexName, docID, doc)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to index document", "index", indexName, "doc_id", docID, "error", err)
		return err
	}

	return nil
}

func (s *InstrumentedEngine) Get(ctx context.Context, indexName, docID string) (*Hit, error) {
	ctx, span := s.startSpan(ctx, "Get",
		attribute.String("search.index", indexName),
		attribute.String("search.doc_id", docID),
	)
	defer span.End()

	logger.L().DebugContext(ctx, "getting document", "index", indexName, "doc_id", docID)

	hit, err := s.next.Get(ctx, indexName, docID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get document", "index", indexName, "doc_id", docID, "error", err)
		return nil, err
	}

	return hit, nil
}

func (s *InstrumentedEngine) Delete(ctx context.Context, indexName, docID string) error {
	ctx, span := s.startSpan(ctx, "Delete",
		attribute.String("search.index", indexName),
		attribute.String("search.doc_id", docID),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "deleting document", "index", indexName, "doc_id", docID)

	err := s.next.Delete(ctx, indexName, docID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete document", "index", indexName, "doc_id", docID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted document", "index", indexName, "doc_id", docID)
	return nil
}

func (s *InstrumentedEngine) Search(ctx context.Context, indexName string, query Query) (*SearchResult, error) {
	ctx, span := s.startSpan(ctx, "Search",
		attribute.String("search.index", indexName),
		attribute.String("search.query", query.Text),
		attribute.Int("search.from", query.From),
		attribute.Int("search.size", query.Size),
	)
	defer span.End()

	logger.L().DebugContext(ctx, "searching", "index", indexName, "query", query.Text)

	start := time.Now()
	result, err := s.next.Search(ctx, indexName, query)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "search failed", "index", indexName, "query", query.Text, "error", err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("search.total_hits", result.Total),
		attribute.Int("search.returned", len(result.Hits)),
	)
	logger.L().DebugContext(ctx, "search complete",
		"index", indexName,
		"total", result.Total,
		"returned", len(result.Hits),
		"duration", duration,
	)
	return result, nil
}

func (s *InstrumentedEngine) Bulk(ctx context.Context, indexName string, ops []BulkOperation) (*BulkResult, error) {
	ctx, span := s.startSpan(ctx, "Bulk",
		attribute.String("search.index", indexName),
		attribute.Int("search.ops_count", len(ops)),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "executing bulk operation", "index", indexName, "count", len(ops))

	start := time.Now()
	result, err := s.next.Bulk(ctx, indexName, ops)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "bulk operation failed", "index", indexName, "error", err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("search.successful", result.Successful),
		attribute.Int("search.failed", result.Failed),
	)
	logger.L().InfoContext(ctx, "bulk operation complete",
		"index", indexName,
		"successful", result.Successful,
		"failed", result.Failed,
		"duration", duration,
	)
	return result, nil
}

func (s *InstrumentedEngine) Refresh(ctx context.Context, indexName string) error {
	ctx, span := s.startSpan(ctx, "Refresh", attribute.String("search.index", indexName))
	defer span.End()

	logger.L().DebugContext(ctx, "refreshing index", "index", indexName)

	err := s.next.Refresh(ctx, indexName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to refresh index", "index", indexName, "error", err)
		return err
	}

	return nil
}

func (s *InstrumentedEngine) Close() error {
	return s.next.Close()
}
