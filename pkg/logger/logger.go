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
	mu            sync.RWMutex
	defaultLogger *slog.Logger
	asyncHandler  *AsyncHandler
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

// Init initializes the global logger and returns it.
//
// Handler stack (outer → inner), built exactly once:
//
//	Sampling → Redact → Trace → Async → JSON/Text
//
// Trace sits outside Async so span attrs are copied onto the record while the
// request context is still available; Async then queues the already-enriched
// record and processes it with a background context.
//
// Call Shutdown before process exit when Async is enabled to flush buffered logs.
func Init(cfg Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				a.Value = slog.StringValue(t.Format(time.RFC3339))
			}
			return a
		},
	}

	var h slog.Handler
	if cfg.Format == "TEXT" {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}

	var async *AsyncHandler
	if cfg.Async {
		async = NewAsyncHandler(h, 4096, true)
		h = async
	}

	// Trace must wrap Async (not the reverse) so correlation attrs are attached
	// before the record is queued.
	h = NewTraceHandler(h)

	if cfg.Redact {
		h = NewRedactHandler(h)
	}

	if cfg.SamplingRate < 1.0 && cfg.SamplingRate > 0.0 {
		h = NewSamplingHandler(h, cfg.SamplingRate)
	}

	l := slog.New(h)
	slog.SetDefault(l)

	mu.Lock()
	defaultLogger = l
	asyncHandler = async
	mu.Unlock()

	return l
}

// L returns the global logger. If Init has not been called, it returns slog.Default().
func L() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if defaultLogger == nil {
		return slog.Default()
	}
	return defaultLogger
}

// Shutdown flushes the AsyncHandler buffer (if Async was enabled in Init) and
// waits for pending records to be written. It is safe to call when Async is
// disabled. Respects ctx cancellation; a cancelled context may leave some
// records unflushed.
func Shutdown(ctx context.Context) error {
	mu.Lock()
	ah := asyncHandler
	asyncHandler = nil
	mu.Unlock()

	if ah == nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		ah.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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

// TraceHandler adds trace_id and span_id to logs from the OpenTelemetry span
// in context. It must sit outside AsyncHandler so attrs are attached before
// the record is queued (Async processes with context.Background).
type TraceHandler struct {
	next slog.Handler
}

// NewTraceHandler wraps next with OpenTelemetry trace/span ID injection.
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
