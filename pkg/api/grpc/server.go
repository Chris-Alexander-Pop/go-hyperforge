package grpc

import (
	"context"
	"net"
	"runtime/debug"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type Config struct {
	Port string `env:"GRPC_PORT" env-default:"9090"`
}

type Server struct {
	srv *grpc.Server
	cfg Config
}

func New(cfg Config) *Server {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(chainUnary(
			otelgrpc.UnaryServerInterceptor(), // OTel Tracing
			RecoveryInterceptor(),             // Panic Recovery
			LoggingInterceptor(),              // Structured Logging
		)),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	}

	srv := grpc.NewServer(opts...)
	reflection.Register(srv)

	return &Server{srv: srv, cfg: cfg}
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

func (s *Server) Stop() {
	s.srv.GracefulStop()
}

// Interceptors (Basic inline for now, or split if grows)

func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := status.Code(err)

		// Log
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

func chainUnary(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		buildChain := func(current grpc.UnaryServerInterceptor, next grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return current(currentCtx, currentReq, info, next)
			}
		}

		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}
		return chain(ctx, req)
	}
}
