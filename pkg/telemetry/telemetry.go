package telemetry

import (
	"context"
	"io"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/telemetry/adapters/noop"
	"github.com/chris-alexander-pop/system-design-library/pkg/telemetry/adapters/stdout"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Provider names for Config.Provider.
const (
	ProviderOTLP   = "otlp"
	ProviderNoop   = "noop"
	ProviderStdout = "stdout"
)

// Config holds configuration for OpenTelemetry.
type Config struct {
	// ServiceName identifies this service in traces.
	ServiceName string `env:"OTEL_SERVICE_NAME" env-default:"unknown-service"`

	// ServiceVersion is the version of this service.
	ServiceVersion string `env:"OTEL_SERVICE_VERSION" env-default:"0.0.1"`

	// Environment is the deployment environment (development, staging, production).
	Environment string `env:"APP_ENV" env-default:"development"`

	// Endpoint is the OTLP collector endpoint (host:port).
	Endpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" env-default:"localhost:4317"`

	// Provider selects the exporter backend: "otlp" (default), "noop", or "stdout".
	// Use "noop" or "stdout" for deterministic tests without a collector.
	Provider string `env:"OTEL_PROVIDER" env-default:"otlp"`

	// SampleRate is the fraction of traces to sample (0.0–1.0). Default 1.0 samples all.
	SampleRate float64 `env:"OTEL_SAMPLE_RATE" env-default:"1.0"`

	// Insecure disables TLS for the OTLP gRPC exporter. Prefer false in production.
	Insecure bool `env:"OTEL_EXPORTER_OTLP_INSECURE" env-default:"false"`

	// StdoutWriter overrides stdout for the stdout provider (tests). Nil uses os.Stdout.
	StdoutWriter io.Writer `json:"-"`
}

// Init initializes the OpenTelemetry tracer provider and returns a shutdown function.
// Prefer Provider "noop" or "stdout" in unit tests so Init never hangs on a collector.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg = normalizeConfig(cfg)

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create resource")
	}

	var (
		tp  *sdktrace.TracerProvider
		err2 error
	)

	switch cfg.Provider {
	case ProviderNoop:
		tp, err2 = noop.NewTracerProvider(res, samplerFor(cfg.SampleRate))
	case ProviderStdout:
		tp, err2 = stdout.NewTracerProvider(res, samplerFor(cfg.SampleRate), cfg.StdoutWriter)
	default: // ProviderOTLP
		tp, err2 = newOTLPProvider(ctx, cfg, res)
	}
	if err2 != nil {
		return nil, err2
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

func normalizeConfig(cfg Config) Config {
	if cfg.Provider == "" {
		cfg.Provider = ProviderOTLP
	}
	cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	if cfg.SampleRate < 0 {
		cfg.SampleRate = 0
	}
	if cfg.SampleRate > 1 {
		cfg.SampleRate = 1
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "unknown-service"
	}
	return cfg
}

func samplerFor(rate float64) sdktrace.Sampler {
	switch {
	case rate <= 0:
		return sdktrace.NeverSample()
	case rate >= 1:
		return sdktrace.AlwaysSample()
	default:
		return sdktrace.TraceIDRatioBased(rate)
	}
}

func newOTLPProvider(ctx context.Context, cfg Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create trace exporter")
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(samplerFor(cfg.SampleRate)),
	), nil
}
