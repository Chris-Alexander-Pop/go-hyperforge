package logger_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

func BenchmarkRedactHandler(b *testing.B) {
	h := slog.NewJSONHandler(io.Discard, nil)
	r := logger.NewRedactHandler(h)
	l := slog.New(r)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.InfoContext(ctx, "User action",
			"user_id", "12345",
			"action", "login",
			"email", "user@example.com",
			"status", "success",
			"description", "User logged in successfully without issues",
			"cc", "1234 5678 1234 5678",
		)
	}
}

func BenchmarkRedactHandler_Clean(b *testing.B) {
	h := slog.NewJSONHandler(io.Discard, nil)
	r := logger.NewRedactHandler(h)
	l := slog.New(r)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.InfoContext(ctx, "User action",
			"user_id", "12345",
			"action", "view_page",
			"page", "dashboard",
			"status", "success",
			"description", "User viewed the dashboard page",
		)
	}
}

// Benchmark to test the fast path optimization when digits are present but no actual CC match
func BenchmarkRedactHandler_CleanWithDigits(b *testing.B) {
	h := slog.NewJSONHandler(io.Discard, nil)
	r := logger.NewRedactHandler(h)
	l := slog.New(r)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.InfoContext(ctx, "User action",
			"user_id", "12345",
			"action", "view_page",
			"page", "dashboard 123",
			"status", "success",
			"description", "User 123 viewed the dashboard page with some numbers",
		)
	}
}
