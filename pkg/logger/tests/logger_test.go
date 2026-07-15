package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

func TestRedactHandler(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	r := logger.NewRedactHandler(h)
	l := slog.New(r)

	l.Info("User login", "email", "john.doe@example.com", "password", "secret123")

	out := buf.String()
	if !strings.Contains(out, "[EMAIL]") {
		t.Error("Email not redacted")
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Error("Password not redacted")
	}
	if strings.Contains(out, "john.doe@example.com") {
		t.Error("Original email leaked")
	}
}

func TestRedactHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	r := logger.NewRedactHandler(h)
	l := slog.New(r).With("password", "leak-me", "token", "abc")

	l.Info("bound attrs")

	out := buf.String()
	if strings.Contains(out, "leak-me") || strings.Contains(out, `"token":"abc"`) {
		t.Errorf("sensitive WithAttrs leaked: %s", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in output: %s", out)
	}
}

func TestSamplingHandler(t *testing.T) {
	// Rate 0.0 -> Log nothing
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	s := logger.NewSamplingHandler(h, 0.0001) // very low
	l := slog.New(s)

	l.Info("Should be dropped")
	if buf.Len() > 0 {
		// Probabilistic, but at 0.0001 it shouldn't log 1 message.
		// Implementation uses atomic counter. 1 % 10000 != 0.
		t.Error("Log should be dropped by sampling")
	}

	// Error always logged
	l.Error("Should be kept")
	if !strings.Contains(buf.String(), "Should be kept") {
		t.Error("Error level should bypass sampling")
	}
}

func TestAsyncHandler(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	a := logger.NewAsyncHandler(h, 100, true)
	l := slog.New(a)

	start := time.Now()
	l.Info("Async message")
	if time.Since(start) > 10*time.Millisecond {
		t.Error("Async log took too long")
	}

	a.Shutdown()
	if !strings.Contains(buf.String(), "Async message") {
		t.Error("Async message not flushed")
	}
}

func TestTraceHandler_WithAsync(t *testing.T) {
	var buf bytes.Buffer
	jsonH := slog.NewJSONHandler(&buf, nil)
	async := logger.NewAsyncHandler(jsonH, 32, false)
	l := slog.New(logger.NewTraceHandler(async))

	tid, _ := trace.TraceIDFromHex("aabbccddeeff00112233445566778899")
	sid, _ := trace.SpanIDFromHex("1122334455667788")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	}))

	l.InfoContext(ctx, "async correlated")
	async.Shutdown()

	out := buf.String()
	if !strings.Contains(out, "aabbccddeeff00112233445566778899") {
		t.Errorf("trace_id missing with Trace outside Async: %s", out)
	}
	if !strings.Contains(out, "1122334455667788") {
		t.Errorf("span_id missing with Trace outside Async: %s", out)
	}
}

func TestShutdown_NoAsync(t *testing.T) {
	_ = logger.Init(logger.Config{
		Level:  "INFO",
		Format: "JSON",
		Async:  false,
		Redact: false,
	})
	if err := logger.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown with Async=false: %v", err)
	}
}
