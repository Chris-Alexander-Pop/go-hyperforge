// Package noop provides a no-op OpenTelemetry tracer provider for tests.
package noop

import (
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// NewTracerProvider returns a TracerProvider that never exports spans.
// Safe for unit tests — Init never contacts a collector.
func NewTracerProvider(res *resource.Resource, sampler sdktrace.Sampler) (*sdktrace.TracerProvider, error) {
	if sampler == nil {
		sampler = sdktrace.NeverSample()
	}
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sampler),
	}
	if res != nil {
		opts = append(opts, sdktrace.WithResource(res))
	}
	return sdktrace.NewTracerProvider(opts...), nil
}
