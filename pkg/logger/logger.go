// Package logger provides structured logging with OpenTelemetry trace correlation.
//
// This package provides:
//   - slog-based structured logging (JSON or TEXT format)
//   - Automatic trace_id and span_id injection from OpenTelemetry context
//   - Global logger accessor via L()
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/logger"
//
//	// Initialize (typically in main)
//	logger.Init(logger.Config{Level: "INFO", Format: "JSON"})
//
//	// Use anywhere via global accessor
//	logger.L().InfoContext(ctx, "message", "key", value)
//	logger.L().ErrorContext(ctx, "failed", "error", err)
package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

// Config holds configuration for the logger.
type Config struct {
	// Level sets the minimum log level: DEBUG, INFO, WARN, ERROR.
	Level string `env:"LOG_LEVEL" env-default:"INFO"`

	// Format sets the output format: JSON or TEXT.
	Format string `env:"LOG_FORMAT" env-default:"JSON"`

	// SamplingRate (0.0 - 1.0). 1.0 = log all.
	SamplingRate float64 `env:"LOG_SAMPLING_RATE" env-default:"1.0"`

	// Async enables non-blocking logging.
	Async bool `env:"LOG_ASYNC" env-default:"true"`

	// Redact enables PII redaction.
	Redact bool `env:"LOG_REDACT" env-default:"true"`
}

// Init initializes the global logger
func Init(cfg Config) *slog.Logger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// standard time format
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				a.Value = slog.StringValue(t.Format(time.RFC3339))
			}
			return a
		},
	}

	if cfg.Format == "TEXT" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	// 1. Trace Injection (always inner layer, close to output)
	handler = NewTraceHandler(handler)

	// 2. Async Buffer (optional)
	if cfg.Async {
		handler = NewAsyncHandler(handler, 4096, true)
	}

	// 3. Sampling (optional, outer layer to drop early)
	if cfg.SamplingRate < 1.0 && cfg.SamplingRate > 0.0 {
		handler = NewSamplingHandler(handler, cfg.SamplingRate)
	}

	// 4. Redaction (optional, expensive so do before output but after sampling?)
	// Actually better to Redact *after* sampling (waste of cpu to redact dropped logs),
	// but *before* Async (to keep buffer clean? Or after async to offload CPU?)
	// Let's do Redact -> Async -> Output. So Redact is BEFORE Async.
	// Order: Sampling (Drop first) -> Redact (Clean) -> Async (Buffer) -> Trace -> Output.
	// Wait, TraceHandler just adds attrs.

	// Updated Order:
	// Sampling -> Redact -> Async -> Trace -> Output

	if cfg.Redact {
		handler = NewRedactHandler(handler)
	}

	// Re-wrap Async if it was added? No, handler is strictly layered.
	// Correct layering:
	// Output = JSONHandler
	// L1 = TraceHandler(Output)
	// L2 = AsyncHandler(L1)
	// L3 = RedactHandler(L2)
	// L4 = SamplingHandler(L3)

	// My previous logic was purely additive, which puts outer layers last.
	// Let's reconstruct cleanly.

	var h slog.Handler = handler // JSON/Text
	h = NewTraceHandler(h)

	if cfg.Async {
		h = NewAsyncHandler(h, 4096, true)
	}

	if cfg.Redact {
		h = NewRedactHandler(h)
	}

	if cfg.SamplingRate < 1.0 && cfg.SamplingRate > 0.0 {
		h = NewSamplingHandler(h, cfg.SamplingRate)
	}

	logger := slog.New(h)
	slog.SetDefault(logger)

	once.Do(func() {
		defaultLogger = logger
	})

	return logger
}

// Global accessor, though we prefer passing logger or using FromContext if we attach it
func L() *slog.Logger {
	if defaultLogger == nil {
		// Fallback if not initialized
		return slog.Default()
	}
	return defaultLogger
}

func parseLevel(level string) slog.Level {
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// TraceHandler adds trace_id and span_id to logs
type TraceHandler struct {
	next slog.Handler
}

func NewTraceHandler(next slog.Handler) *TraceHandler {
	return &TraceHandler{next: next}
}

func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return h.next.Handle(ctx, r)
}

func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{next: h.next.WithAttrs(attrs)}
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{next: h.next.WithGroup(name)}
}
