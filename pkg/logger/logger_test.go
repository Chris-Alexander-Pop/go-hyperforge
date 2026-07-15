package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// TestInit_HandlerStackOnce verifies Init builds Sampling→Redact→Trace→Async→output
// exactly once (no double Async / Trace wrapping).
func TestInit_HandlerStackOnce(t *testing.T) {
	l := Init(Config{
		Level:        "INFO",
		Format:       "JSON",
		Async:        true,
		Redact:       true,
		SamplingRate: 0.5,
	})
	defer func() { _ = Shutdown(context.Background()) }()

	h := l.Handler()

	sh, ok := h.(*SamplingHandler)
	if !ok {
		t.Fatalf("outer handler: got %T, want *SamplingHandler", h)
	}

	rh, ok := sh.next.(*RedactHandler)
	if !ok {
		t.Fatalf("after Sampling: got %T, want *RedactHandler", sh.next)
	}

	th, ok := rh.next.(*TraceHandler)
	if !ok {
		t.Fatalf("after Redact: got %T, want *TraceHandler", rh.next)
	}

	ah, ok := th.next.(*AsyncHandler)
	if !ok {
		t.Fatalf("after Trace: got %T, want *AsyncHandler", th.next)
	}

	// Inner must be the JSON/text handler — not another Async or Trace.
	switch ah.next.(type) {
	case *AsyncHandler, *TraceHandler, *RedactHandler, *SamplingHandler:
		t.Fatalf("Async next should be output handler, got %T (double-wrap)", ah.next)
	}
}

// TestInit_TraceOutsideAsync_Correlation ensures OTEL span IDs survive Async=true.
func TestInit_TraceOutsideAsync_Correlation(t *testing.T) {
	var buf bytes.Buffer
	jsonH := slog.NewJSONHandler(&buf, nil)
	async := NewAsyncHandler(jsonH, 64, false)
	traceH := NewTraceHandler(async)
	l := slog.New(traceH)

	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	if err != nil {
		t.Fatal(err)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	l.InfoContext(ctx, "correlated")
	async.Shutdown()

	out := buf.String()
	if !strings.Contains(out, `"trace_id":"0102030405060708090a0b0c0d0e0f10"`) {
		t.Fatalf("missing trace_id in async output: %s", out)
	}
	if !strings.Contains(out, `"span_id":"0102030405060708"`) {
		t.Fatalf("missing span_id in async output: %s", out)
	}
}

// TestShutdown_FlushesAsync verifies package Shutdown drains the Init async buffer.
func TestShutdown_FlushesAsync(t *testing.T) {
	// Use a custom stack so we can capture output (Init writes to stdout).
	var buf bytes.Buffer
	jsonH := slog.NewJSONHandler(&buf, nil)
	async := NewAsyncHandler(jsonH, 64, false)

	mu.Lock()
	asyncHandler = async
	mu.Unlock()

	l := slog.New(NewTraceHandler(async))
	l.Info("must flush")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if !strings.Contains(buf.String(), "must flush") {
		t.Fatalf("Shutdown did not flush: %q", buf.String())
	}
}

// TestRedactHandler_WithAttrsRedactsBound ensures logger.With passwords are redacted.
func TestRedactHandler_WithAttrsRedactsBound(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	r := NewRedactHandler(h)
	l := slog.New(r).With("password", "super-secret", "user", "alice")

	l.Info("login")

	out := buf.String()
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if m["password"] != "[REDACTED]" {
		t.Errorf("bound password not redacted: %v", m["password"])
	}
	if strings.Contains(out, "super-secret") {
		t.Error("password value leaked via WithAttrs")
	}
	if m["user"] != "alice" {
		t.Errorf("non-sensitive attr changed: %v", m["user"])
	}
}
