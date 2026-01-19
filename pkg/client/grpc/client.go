package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Config struct {
	Target  string        `env:"CLIENT_GRPC_TARGET" env-required:"true"`
	Timeout time.Duration `env:"CLIENT_TIMEOUT" env-default:"5s"`

	// TLS settings
	TLSEnabled bool   `env:"CLIENT_TLS_ENABLED" env-default:"false"`
	CAFile     string `env:"CLIENT_TLS_CA_FILE"`
	CertFile   string `env:"CLIENT_TLS_CERT_FILE"`
	KeyFile    string `env:"CLIENT_TLS_KEY_FILE"`
	Insecure   bool   `env:"CLIENT_TLS_INSECURE" env-default:"false"`

	// Auth
	BearerToken string `env:"CLIENT_BEARER_TOKEN"`

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
	var opts []grpc.DialOption

	// 1. Credentials (TLS or Insecure)
	if cfg.TLSEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.Insecure,
		}
		if cfg.CAFile != "" {
			caCert, err := os.ReadFile(cfg.CAFile)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read CA file")
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caCertPool
		}
		if cfg.CertFile != "" && cfg.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load client cert")
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// 2. Auth Credentials
	if cfg.BearerToken != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(tokenAuth{token: cfg.BearerToken}))
	}

	// 3. OTel and Logging
	opts = append(opts,
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithUnaryInterceptor(LoggingUnaryInterceptor),
		grpc.WithStreamInterceptor(LoggingStreamInterceptor),
	)

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

// LoggingUnaryInterceptor logs gRPC unary requests.
func LoggingUnaryInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	duration := time.Since(start)

	entry := logger.L().With(
		"method", method,
		"duration_ms", duration.Milliseconds(),
		"type", "unary",
	)

	if err != nil {
		entry.Error("grpc request failed", "error", err)
	} else {
		entry.Info("grpc request completed")
	}

	return err
}

// LoggingStreamInterceptor logs gRPC stream requests.
func LoggingStreamInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	start := time.Now()
	stream, err := streamer(ctx, desc, cc, method, opts...)
	duration := time.Since(start)

	entry := logger.L().With(
		"method", method,
		"duration_ms", duration.Milliseconds(),
		"type", "stream",
	)

	if err != nil {
		entry.Error("grpc stream creation failed", "error", err)
	} else {
		entry.Info("grpc stream started")
	}

	return stream, err
}

// tokenAuth implements credentials.PerRPCCredentials for Bearer tokens.
type tokenAuth struct {
	token string
}

func (t tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (t tokenAuth) RequireTransportSecurity() bool {
	return true // Ideally should be true, but depends on config
}
