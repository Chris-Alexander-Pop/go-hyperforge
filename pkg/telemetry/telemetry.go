// Package telemetry provides OpenTelemetry tracing initialization.
//
// This package sets up the OpenTelemetry tracer provider with OTLP export.
// Traces are automatically correlated with logs via pkg/logger.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/telemetry"
//
//	shutdown, err := telemetry.Init(telemetry.Config{
//		ServiceName: "my-service",
//		Endpoint:    "localhost:4317",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer shutdown(context.Background())
package telemetry

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Config holds configuration for OpenTelemetry.
type Config struct {
	// ServiceName identifies this service in traces.
	ServiceName string `env:"OTEL_SERVICE_NAME" env-default:"unknown-service"`

	// ServiceVersion is the version of this service.
	ServiceVersion string `env:"OTEL_SERVICE_VERSION" env-default:"0.0.1"`

	// Environment is the deployment environment (development, staging, production).
	Environment string `env:"APP_ENV" env-default:"development"`

	// Endpoint is the OTLP collector endpoint.
	Endpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" env-default:"localhost:4317"`
}

// Init initializes the OpenTelemetry tracer provider and returns a shutdown function
func Init(cfg Config) (func(context.Context) error, error) {
	ctx := context.Background()

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

	// Set up OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(), // Use WithInsecure for now; in prod, configure TLS
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create trace exporter")
	}

	// Register TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample all traces for now
	)
	otel.SetTracerProvider(tp)

	// Set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}
