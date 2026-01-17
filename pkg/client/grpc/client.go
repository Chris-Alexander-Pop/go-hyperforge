package grpc

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Config struct {
	Target  string        `env:"CLIENT_GRPC_TARGET" env-required:"true"`
	Timeout time.Duration `env:"CLIENT_TIMEOUT" env-default:"5s"`

	// Circuit breaker settings
	CircuitBreakerEnabled   bool          `env:"CLIENT_GRPC_CB_ENABLED" env-default:"true"`
	CircuitBreakerThreshold int64         `env:"CLIENT_GRPC_CB_THRESHOLD" env-default:"5"`
	CircuitBreakerTimeout   time.Duration `env:"CLIENT_GRPC_CB_TIMEOUT" env-default:"30s"`

	// Retry settings
	RetryEnabled     bool          `env:"CLIENT_GRPC_RETRY_ENABLED" env-default:"true"`
	RetryMaxAttempts int           `env:"CLIENT_GRPC_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"CLIENT_GRPC_RETRY_BACKOFF" env-default:"100ms"`
}

// New creates a robust gRPC connection with optional resilience features.
func New(ctx context.Context, cfg Config) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}

	// Add circuit breaker interceptor if enabled
	if cfg.CircuitBreakerEnabled {
		cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "grpc-client-" + cfg.Target,
			FailureThreshold: cfg.CircuitBreakerThreshold,
			SuccessThreshold: 2,
			Timeout:          cfg.CircuitBreakerTimeout,
		})

		opts = append(opts,
			grpc.WithUnaryInterceptor(CircuitBreakerUnaryInterceptor(cb)),
			grpc.WithStreamInterceptor(CircuitBreakerStreamInterceptor(cb)),
		)
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Target, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// CircuitBreakerUnaryInterceptor creates a unary client interceptor with circuit breaker.
func CircuitBreakerUnaryInterceptor(cb *resilience.CircuitBreaker) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return cb.Execute(ctx, func(ctx context.Context) error {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err != nil {
				// Only count certain errors as failures
				if shouldCountAsFailure(err) {
					return err
				}
				// Return error but don't count as circuit breaker failure
				return &nonCircuitError{err: err}
			}
			return nil
		})
	}
}

// CircuitBreakerStreamInterceptor creates a stream client interceptor with circuit breaker.
func CircuitBreakerStreamInterceptor(cb *resilience.CircuitBreaker) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// Check circuit breaker before creating stream
		if cb.State() == resilience.StateOpen {
			return nil, resilience.ErrCircuitOpen
		}

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			if shouldCountAsFailure(err) {
				cb.Execute(ctx, func(ctx context.Context) error { return err })
			}
			return nil, err
		}

		return stream, nil
	}
}

// shouldCountAsFailure determines if an error should count toward the circuit breaker threshold.
func shouldCountAsFailure(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return true // Unknown errors count as failures
	}

	// Only infrastructure-level errors should trigger circuit breaker
	switch st.Code() {
	case codes.Unavailable,
		codes.DeadlineExceeded,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.Internal:
		return true
	default:
		// Business logic errors (NotFound, InvalidArgument, etc.) don't trigger CB
		return false
	}
}

// nonCircuitError wraps an error that shouldn't be counted by circuit breaker.
type nonCircuitError struct {
	err error
}

func (e *nonCircuitError) Error() string {
	return e.err.Error()
}

func (e *nonCircuitError) Unwrap() error {
	return e.err
}

// RetryUnaryInterceptor creates a unary client interceptor with retry logic.
func RetryUnaryInterceptor(cfg resilience.RetryConfig) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return resilience.Retry(ctx, cfg, func(ctx context.Context) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		})
	}
}
