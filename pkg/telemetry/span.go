package telemetry

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordError records err on the span and marks the span status as Error.
// No-op when err is nil or span is invalid/nil-safe via the OpenTelemetry API.
func RecordError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetStatus sets the span status. Empty description is allowed for Ok/Unset.
func SetStatus(span trace.Span, code codes.Code, description string) {
	if span == nil {
		return
	}
	span.SetStatus(code, description)
}
