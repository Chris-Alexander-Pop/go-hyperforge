package logger_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

func BenchmarkRedactHandler(b *testing.B) {
	// Discard output
	h := slog.NewJSONHandler(io.Discard, nil)
	r := logger.NewRedactHandler(h)
	l := slog.New(r)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Log a mix of clean and sensitive data
		l.InfoContext(ctx, "User action",
			"user_id", "12345",
			"action", "login",
			"email", "user@example.com", // Needs redaction
			"status", "success",
			"description", "User logged in successfully without issues", // Clean long string
			"cc", "1234 5678 1234 5678", // Needs redaction
		)
	}
}

func BenchmarkRedactHandler_Clean(b *testing.B) {
	// Discard output
	h := slog.NewJSONHandler(io.Discard, nil)
	r := logger.NewRedactHandler(h)
	l := slog.New(r)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Log only clean data
		l.InfoContext(ctx, "User action",
			"user_id", "12345",
			"action", "view_page",
			"page", "dashboard",
			"status", "success",
			"description", "User viewed the dashboard page",
		)
	}
}
