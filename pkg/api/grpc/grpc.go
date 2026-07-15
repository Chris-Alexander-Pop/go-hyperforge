package grpc

import (
	"context"
	"net"
	"runtime/debug"
	"strings"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
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
			StreamErrorInterceptor(),
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

// StreamErrorInterceptor maps stream handler errors via pkg/errors.GRPCStatus.
func StreamErrorInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		if err == nil {
			return nil
		}
		if pkgerrors.Code(err) != "" {
			return pkgerrors.GRPCStatus(err).Err()
		}
		if st, ok := status.FromError(err); ok {
			return st.Err()
		}
		return pkgerrors.GRPCStatus(err).Err()
	}
}

// TokenVerifier checks a bearer token and returns subject and roles.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (subject string, roles []string, err error)
}

type authContextKey string

const (
	ContextKeySubject authContextKey = "grpc.auth.subject"
	ContextKeyRoles   authContextKey = "grpc.auth.roles"
)

// AuthInterceptor is a unary interceptor that requires a Bearer token in
// metadata key "authorization" and attaches subject/roles to the context.
func AuthInterceptor(verifier TokenVerifier) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, err := authenticateContext(ctx, verifier)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamAuthInterceptor is the streaming counterpart of AuthInterceptor.
func StreamAuthInterceptor(verifier TokenVerifier) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := authenticateContext(ss.Context(), verifier)
		if err != nil {
			return err
		}
		return handler(srv, &authenticatedStream{ServerStream: ss, ctx: ctx})
	}
}

type authenticatedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedStream) Context() context.Context { return s.ctx }

func authenticateContext(ctx context.Context, verifier TokenVerifier) (context.Context, error) {
	if verifier == nil {
		return ctx, status.Error(codes.Internal, "auth verifier is nil")
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return ctx, status.Error(codes.Unauthenticated, "missing authorization")
	}
	token := vals[0]
	if len(token) > 7 && strings.EqualFold(token[:7], "bearer ") {
		token = token[7:]
	}
	sub, roles, err := verifier.Verify(ctx, token)
	if err != nil {
		return ctx, pkgerrors.GRPCStatus(pkgerrors.Unauthorized("invalid token", err)).Err()
	}
	ctx = context.WithValue(ctx, ContextKeySubject, sub)
	ctx = context.WithValue(ctx, ContextKeyRoles, roles)
	return ctx, nil
}

// GetSubject returns the authenticated subject from context (empty if unset).
func GetSubject(ctx context.Context) string {
	s, _ := ctx.Value(ContextKeySubject).(string)
	return s
}

// GetRoles returns authenticated roles from context (nil if unset).
func GetRoles(ctx context.Context) []string {
	r, _ := ctx.Value(ContextKeyRoles).([]string)
	return r
}
