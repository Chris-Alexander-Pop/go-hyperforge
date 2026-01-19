/*
Package telemetry provides OpenTelemetry tracing initialization.

This package sets up the OpenTelemetry tracer provider with OTLP export.
Traces are automatically correlated with logs via pkg/logger.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/telemetry"

	shutdown, err := telemetry.Init(telemetry.Config{
		ServiceName: "my-service",
		Endpoint:    "localhost:4317",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(context.Background())
*/
package telemetry
