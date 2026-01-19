package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestRedactHandler(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	r := NewRedactHandler(h)
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

func TestSamplingHandler(t *testing.T) {
	// Rate 0.0 -> Log nothing
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	s := NewSamplingHandler(h, 0.0001) // very low
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
	// We need a thread-safe buffer or pipe
	var buf bytes.Buffer
	// slog.Handler is not concurrent safe usually when writing to bytes.Buffer.
	// But JSONHandler writes are atomic enough? No.
	// We'll use a channel-based mock handler or just assume JSONHandler to stdout works.
	// Let's verify shutdown logic.

	h := slog.NewJSONHandler(&buf, nil) // Not safe but acceptable for single test sequence
	a := NewAsyncHandler(h, 100, true)
	l := slog.New(a)

	start := time.Now()
	l.Info("Async message")
	// Should complete instantly
	if time.Since(start) > 10*time.Millisecond {
		t.Error("Async log took too long")
	}

	a.Shutdown()
	// Now buffering should flush
	if !strings.Contains(buf.String(), "Async message") {
		t.Error("Async message not flushed")
	}
}
