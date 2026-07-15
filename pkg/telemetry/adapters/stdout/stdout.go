// Package stdout provides a stdout OpenTelemetry trace/metric exporter for local/debug use.
package stdout

import (
	"io"
	"os"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// NewTracerProvider returns a TracerProvider that writes spans to w (or os.Stdout).
func NewTracerProvider(res *resource.Resource, sampler sdktrace.Sampler, w io.Writer) (*sdktrace.TracerProvider, error) {
	if w == nil {
		w = os.Stdout
	}
	if sampler == nil {
		sampler = sdktrace.AlwaysSample()
	}

	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(w),
		stdouttrace.WithoutTimestamps(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create stdout trace exporter")
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sampler),
	}
	if res != nil {
		opts = append(opts, sdktrace.WithResource(res))
	}
	return sdktrace.NewTracerProvider(opts...), nil
}

// NewMeterProvider returns a MeterProvider that writes metrics to w (or os.Stdout).
func NewMeterProvider(res *resource.Resource, w io.Writer) (*sdkmetric.MeterProvider, error) {
	if w == nil {
		w = os.Stdout
	}
	exporter, err := stdoutmetric.New(stdoutmetric.WithWriter(w))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create stdout metric exporter")
	}
	opts := []sdkmetric.Option{
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	}
	if res != nil {
		opts = append(opts, sdkmetric.WithResource(res))
	}
	return sdkmetric.NewMeterProvider(opts...), nil
}
