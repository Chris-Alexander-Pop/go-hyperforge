/*
Package telemetry provides OpenTelemetry tracing and metrics initialization.

Supports OTLP export plus noop/stdout providers for deterministic tests.
Traces are correlated with logs via pkg/logger. MeterProvider is initialized
alongside the TracerProvider unless Config.DisableMetrics is set.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/telemetry"

	shutdown, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName: "my-service",
		Provider:    telemetry.ProviderNoop, // or ProviderStdout / ProviderOTLP
		SampleRate:  1.0,
		Insecure:    true, // only for local OTLP without TLS
	})
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(context.Background())

	counter, _ := telemetry.Meter("my-service").Int64Counter("requests")
	counter.Add(ctx, 1)

	// Shared helpers for instrumented wrappers:
	// telemetry.RecordError(span, err)
	// telemetry.SetStatus(span, codes.Error, msg)
*/
package telemetry
