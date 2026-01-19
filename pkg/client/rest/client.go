package rest

import (
	"context"
	"net/http"
	"time"

	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Config struct {
	Timeout   time.Duration `env:"CLIENT_TIMEOUT" env-default:"30s"`
	Retries   int           `env:"CLIENT_RETRIES" env-default:"3"`
	UserAgent string        `env:"CLIENT_USER_AGENT" env-default:"system-design-library-client"`

	// Auth
	BearerToken string `env:"CLIENT_BEARER_TOKEN"`
	APIKey      string `env:"CLIENT_API_KEY"`

	// TLS settings
	TLSEnabled bool   `env:"CLIENT_TLS_ENABLED" env-default:"false"`
	CAFile     string `env:"CLIENT_TLS_CA_FILE"`
	CertFile   string `env:"CLIENT_TLS_CERT_FILE"`
	KeyFile    string `env:"CLIENT_TLS_KEY_FILE"`
	Insecure   bool   `env:"CLIENT_TLS_INSECURE" env-default:"false"`

	// Circuit breaker settings
	CircuitBreakerEnabled   bool          `env:"CLIENT_CB_ENABLED" env-default:"true"`
	CircuitBreakerThreshold int64         `env:"CLIENT_CB_THRESHOLD" env-default:"5"`
	CircuitBreakerTimeout   time.Duration `env:"CLIENT_CB_TIMEOUT" env-default:"30s"`
}

// Client wraps http.Client with resilience features.
type Client struct {
	httpClient     *http.Client
	circuitBreaker *resilience.CircuitBreaker
	config         Config
}

// New creates a robust HTTP client with Retries, Circuit Breaker, and OTel Tracing
func New(cfg Config) (*Client, error) {
	// 1. Retryable Client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = cfg.Retries
	retryClient.HTTPClient.Timeout = cfg.Timeout
	retryClient.Logger = nil

	// 2. Configure Transport (TLS -> Logging -> OTel -> Base)
	baseTransport := retryClient.HTTPClient.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	// Handle TLS
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

		// Clone default transport to avoid modifying global
		if t, ok := baseTransport.(*http.Transport); ok {
			t2 := t.Clone()
			t2.TLSClientConfig = tlsConfig
			baseTransport = t2
		} else {
			// Fallback if not a standard http.Transport
			baseTransport = &http.Transport{
				TLSClientConfig: tlsConfig,
			}
		}
	}

	// Add Logging
	loggingTransport := &LoggingTransport{
		Next: baseTransport,
	}

	// Add OTel
	otelTransport := otelhttp.NewTransport(loggingTransport)
	retryClient.HTTPClient.Transport = otelTransport

	// 3. Create standard client
	stdClient := retryClient.StandardClient()

	client := &Client{
		httpClient: stdClient,
		config:     cfg,
	}

	// 4. Create circuit breaker if enabled
	if cfg.CircuitBreakerEnabled {
		client.circuitBreaker = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "rest-client",
			FailureThreshold: cfg.CircuitBreakerThreshold,
			SuccessThreshold: 2,
			Timeout:          cfg.CircuitBreakerTimeout,
		})
	}

	return client, nil
}

// LoggingTransport logs requests and responses
type LoggingTransport struct {
	Next http.RoundTripper
}

func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	resp, err := t.Next.RoundTrip(req)

	duration := time.Since(start)

	// Create log entry with builder pattern
	entry := logger.L().With(
		"method", req.Method,
		"url", req.URL.String(),
		"duration_ms", duration.Milliseconds(),
	)

	if err != nil {
		entry.Error("http request failed", "error", err)
		return nil, err
	}

	entry.Info("http request completed", "status", resp.StatusCode)

	return resp, nil
}

// Do executes the request with circuit breaker protection.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.circuitBreaker == nil {
		return c.httpClient.Do(req)
	}

	var resp *http.Response
	err := c.circuitBreaker.Execute(req.Context(), func(ctx context.Context) error {
		var err error
		reqWithCtx := req.WithContext(ctx)
		resp, err = c.httpClient.Do(reqWithCtx)

		// Only count server errors (5xx) as failures for circuit breaker
		if err == nil && resp != nil && resp.StatusCode >= 500 {
			return &serverError{statusCode: resp.StatusCode}
		}
		return err
	})

	// Unwrap server error - we still want to return the response
	if _, ok := err.(*serverError); ok {
		return resp, nil
	}

	return resp, err
}

// Get performs a GET request with circuit breaker protection.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Inject Auth Headers
	if c.config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	} else if c.config.APIKey != "" {
		req.Header.Set("X-API-Key", c.config.APIKey)
	}

	if c.config.UserAgent != "" {
		req.Header.Set("User-Agent", c.config.UserAgent)
	}

	return c.Do(req)
}

// HTTPClient returns the underlying http.Client for direct use.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// CircuitBreakerState returns the current circuit breaker state, or empty if disabled.
func (c *Client) CircuitBreakerState() resilience.State {
	if c.circuitBreaker == nil {
		return ""
	}
	return c.circuitBreaker.State()
}

// serverError is used internally to track server errors for circuit breaker.
type serverError struct {
	statusCode int
}

func (e *serverError) Error() string {
	return "server error"
}

// NewSimple creates a simple HTTP client without circuit breaker (backward compatible).
func NewSimple(cfg Config) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = cfg.Retries
	retryClient.HTTPClient.Timeout = cfg.Timeout
	retryClient.Logger = nil

	baseTransport := retryClient.HTTPClient.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	otelTransport := otelhttp.NewTransport(baseTransport)
	retryClient.HTTPClient.Transport = otelTransport

	return retryClient.StandardClient()
}
