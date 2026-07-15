package grpc

import (
	"context"
	"net"
	"runtime/debug"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type Config struct {
	Port string `env:"GRPC_PORT" env-default:"9090"`
}

type Server struct {
	srv    *grpc.Server
	cfg    Config
	health *health.Server
}

func New(cfg Config) *Server {
	opts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			RecoveryInterceptor(),
			ErrorInterceptor(),
			LoggingInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			StreamRecoveryInterceptor(),
		),
	}

	srv := grpc.NewServer(opts...)
	reflection.Register(srv)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	return &Server{srv: srv, cfg: cfg, health: hs}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", ":"+s.cfg.Port)
	if err != nil {
		return err
	}
	logger.L().InfoContext(context.Background(), "starting grpc server", "port", s.cfg.Port)
	return s.srv.Serve(lis)
}

func (s *Server) GRPC() *grpc.Server {
	return s.srv
}

// Health returns the registered gRPC health server for status updates.
func (s *Server) Health() *health.Server {
	return s.health
}

func (s *Server) Stop() {
	if s.health != nil {
		s.health.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}
	s.srv.GracefulStop()
}

// ErrorInterceptor maps pkg/errors.AppError values to gRPC status codes via GRPCStatus.
func ErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}
		// Prefer AppError → full code map.
		if pkgerrors.Code(err) != "" {
			return resp, pkgerrors.GRPCStatus(err).Err()
		}
		// Preserve existing gRPC status errors.
		if st, ok := status.FromError(err); ok {
			return resp, st.Err()
		}
		return resp, pkgerrors.GRPCStatus(err).Err()
	}
}

func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)
		code := status.Code(err)

		logFn := logger.L().InfoContext
		if err != nil {
			logFn = logger.L().ErrorContext
		}

		logFn(ctx, "grpc request",
			"method", info.FullMethod,
			"code", code.String(),
			"latency", duration,
			"error", err,
		)

		return resp, err
	}
}

func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = status.Errorf(codes.Internal, "panic: %v", r)
				logger.L().ErrorContext(ctx, "grpc panic recovered",
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor recovers from panics in streaming RPCs.
func StreamRecoveryInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = status.Errorf(codes.Internal, "panic: %v", r)
				logger.L().ErrorContext(ss.Context(), "grpc stream panic recovered",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		return handler(srv, ss)
	}
}
