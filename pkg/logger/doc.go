// Package logger provides structured logging with OpenTelemetry trace correlation.
//
// # Initialization
//
// Call Init once at process start (typically in main). Until Init runs, L() falls
// back to slog.Default() and does not apply sampling, redaction, async buffering,
// or trace correlation.
//
//	logger.Init(logger.Config{Level: "INFO", Format: "JSON"})
//	defer logger.Shutdown(context.Background())
//
// # Shutdown
//
// When Async is enabled (the default), call Shutdown before exit to flush the
// AsyncHandler buffer. Skipping Shutdown can lose recent log lines.
//
// # Handler order
//
// Init builds a single handler stack (outer → inner):
//
//	Sampling → Redact → Trace → Async → JSON/Text
//
// Trace sits outside Async so trace_id and span_id are copied onto the record
// while the request context is live. Async then queues the enriched record.
//
// # Usage
//
//	logger.L().InfoContext(ctx, "message", "key", value)
//	logger.L().ErrorContext(ctx, "failed", "error", err)
//
// Bound attributes via logger.With are redacted when Redact is enabled.
package logger
