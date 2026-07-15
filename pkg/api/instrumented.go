package api

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedServer wraps Server with logging and tracing
type InstrumentedServer struct {
	next   Server
	tracer trace.Tracer
}

func NewInstrumentedServer(next Server) *InstrumentedServer {
	return &InstrumentedServer{
		next:   next,
		tracer: otel.Tracer("pkg/api"),
	}
}

func (s *InstrumentedServer) Start() error {
	ctx := context.Background() // Start usually doesn't take context, creating one for logging
	logger.L().InfoContext(ctx, "starting api server")

	// We can't easily trace Start() as it blocks, but we can log
	err := s.next.Start()
	if err != nil {
		logger.L().ErrorContext(ctx, "api server failed", "error", err)
	}
	return err
}

func (s *InstrumentedServer) Shutdown(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "api.Shutdown")
	defer span.End()

	logger.L().InfoContext(ctx, "shutting down api server")

	start := time.Now()
	err := s.next.Shutdown(ctx)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to shutdown api server", "error", err, "duration", duration)
		return err
	}

	logger.L().InfoContext(ctx, "api server shutdown complete", "duration", duration)
	return nil
}
