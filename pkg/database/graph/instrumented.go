package graph

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type InstrumentedGraph struct {
	next   Interface
	tracer trace.Tracer
}

func NewInstrumented(next Interface) *InstrumentedGraph {
	return &InstrumentedGraph{
		next:   next,
		tracer: otel.Tracer("pkg/database/graph"),
	}
}

func (i *InstrumentedGraph) AddVertex(ctx context.Context, v *Vertex) error {
	ctx, span := i.tracer.Start(ctx, "graph.AddVertex", trace.WithAttributes(
		attribute.String("label", v.Label),
		attribute.String("id", v.ID),
	))
	defer span.End()

	err := i.next.AddVertex(ctx, v)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "graph add vertex failed", "error", err, "id", v.ID)
	}
	return err
}

func (i *InstrumentedGraph) AddEdge(ctx context.Context, e *Edge) error {
	ctx, span := i.tracer.Start(ctx, "graph.AddEdge", trace.WithAttributes(
		attribute.String("label", e.Label),
		attribute.String("from", e.FromID),
		attribute.String("to", e.ToID),
	))
	defer span.End()

	err := i.next.AddEdge(ctx, e)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "graph add edge failed", "error", err, "label", e.Label)
	}
	return err
}

func (i *InstrumentedGraph) GetVertex(ctx context.Context, id string) (*Vertex, error) {
	ctx, span := i.tracer.Start(ctx, "graph.GetVertex", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	v, err := i.next.GetVertex(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "graph get vertex failed", "error", err, "id", id)
	}
	return v, err
}

func (i *InstrumentedGraph) GetNeighbors(ctx context.Context, vertexID string, edgeLabel string, direction string) ([]*Vertex, error) {
	ctx, span := i.tracer.Start(ctx, "graph.GetNeighbors", trace.WithAttributes(
		attribute.String("vertex_id", vertexID),
		attribute.String("edge_label", edgeLabel),
	))
	defer span.End()

	neighbors, err := i.next.GetNeighbors(ctx, vertexID, edgeLabel, direction)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "graph get neighbors failed", "error", err, "vertex_id", vertexID)
	} else {
		span.SetAttributes(attribute.Int("result_count", len(neighbors)))
	}
	return neighbors, err
}

func (i *InstrumentedGraph) Query(ctx context.Context, query string, args map[string]interface{}) (interface{}, error) {
	ctx, span := i.tracer.Start(ctx, "graph.Query", trace.WithAttributes(
		attribute.String("query", query),
	))
	defer span.End()

	res, err := i.next.Query(ctx, query, args)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "graph query failed", "error", err, "query", query)
	}
	return res, err
}

func (i *InstrumentedGraph) Close() error {
	return i.next.Close()
}
