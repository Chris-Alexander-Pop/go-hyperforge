package network

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// serverRunner is satisfied by TCPServer and UDPServer.
type serverRunner interface {
	ListenAndServe(ctx context.Context) error
}

// InstrumentedServer wraps a TCP or UDP server with logging and tracing around ListenAndServe.
type InstrumentedServer struct {
	next   serverRunner
	name   string
	addr   string
	tracer trace.Tracer
}

// NewInstrumentedTCPServer wraps a TCPServer with instrumentation.
func NewInstrumentedTCPServer(server *TCPServer) *InstrumentedServer {
	return &InstrumentedServer{
		next:   server,
		name:   "tcp",
		addr:   server.cfg.Addr,
		tracer: otel.Tracer("pkg/network"),
	}
}

// NewInstrumentedUDPServer wraps a UDPServer with instrumentation.
func NewInstrumentedUDPServer(server *UDPServer) *InstrumentedServer {
	return &InstrumentedServer{
		next:   server,
		name:   "udp",
		addr:   server.cfg.Addr,
		tracer: otel.Tracer("pkg/network"),
	}
}

// ListenAndServe starts the underlying server with a root span for the listen lifecycle.
func (s *InstrumentedServer) ListenAndServe(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "network."+s.name+".ListenAndServe", trace.WithAttributes(
		attribute.String("network.protocol", s.name),
		attribute.String("network.addr", s.addr),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "starting instrumented network server", "protocol", s.name, "addr", s.addr)

	err := s.next.ListenAndServe(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "network server failed", "protocol", s.name, "error", err)
		return err
	}
	return nil
}
