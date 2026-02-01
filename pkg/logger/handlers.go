package logger

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync/atomic"
)

// --- Sampling Handler ---

// SamplingHandler drops records based on a sampling rate (0.0 to 1.0).
// 1.0 = log everything, 0.0 = log nothing.
// Errors are ALWAYS logged regardless of sampling rate.
type SamplingHandler struct {
	next slog.Handler
	rate float64
	// Simple counter for deterministic sampling (or use rand)
	// For high perf, atomic counter % N is faster than rand
	counter atomic.Uint64
	n       uint64 // rate * 100 or similar explanation
}

func NewSamplingHandler(next slog.Handler, rate float64) *SamplingHandler {
	if rate >= 1.0 {
		return &SamplingHandler{next: next, rate: 1.0}
	}
	// Inverse: if rate is 0.01 (1%), we log every 100th
	// Simple implementation: use 10000 scale
	return &SamplingHandler{
		next: next,
		rate: rate,
		n:    uint64(1.0 / rate),
	}
}

func (h *SamplingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Always enable errors, otherwise check parent
	if level >= slog.LevelError {
		return h.next.Enabled(ctx, level)
	}
	return h.next.Enabled(ctx, level)
}

func (h *SamplingHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError || h.rate >= 1.0 {
		return h.next.Handle(ctx, r)
	}

	// Probabilistic check
	// Atomic increment is thread safe
	c := h.counter.Add(1)
	if c%h.n == 0 {
		return h.next.Handle(ctx, r)
	}
	return nil
}

func (h *SamplingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SamplingHandler{next: h.next.WithAttrs(attrs), rate: h.rate, n: h.n}
}

func (h *SamplingHandler) WithGroup(name string) slog.Handler {
	return &SamplingHandler{next: h.next.WithGroup(name), rate: h.rate, n: h.n}
}

// --- Async Handler ---

// AsyncHandler processes logs in a background goroutine to avoid blocking the caller.
// Uses a buffered channel (Ring Buffer concept).
type AsyncHandler struct {
	next       slog.Handler
	buffer     chan slog.Record
	done       chan struct{}
	dropOnFull bool // If true, drop logs when buffer full. If false, block (backpressure)
}

func NewAsyncHandler(next slog.Handler, bufferSize int, dropOnFull bool) *AsyncHandler {
	h := &AsyncHandler{
		next:       next,
		buffer:     make(chan slog.Record, bufferSize),
		done:       make(chan struct{}),
		dropOnFull: dropOnFull,
	}
	go h.process()
	return h
}

func (h *AsyncHandler) Handle(ctx context.Context, r slog.Record) error {
	// Clone record because it might be reused by caller (slog optimization)
	// We need own copy for async
	r2 := r.Clone()

	if h.dropOnFull {
		select {
		case h.buffer <- r2:
		default:
			// Buffer full, drop
			// In metric world, increment "dropped_logs" counter
		}
	} else {
		h.buffer <- r2
	}
	return nil
}

func (h *AsyncHandler) process() {
	defer close(h.done)
	for r := range h.buffer {
		// Context is background because original context might be dead
		_ = h.next.Handle(context.Background(), r)
	}
}

func (h *AsyncHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *AsyncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Creating new AsyncHandler for WithAttrs is complex because it spawns new goroutine/channel
	// USUALLY, Async is the LAST wrapper before output (or FIRST).
	// If it's first, WithAttrs creates valid Record attributes but same Handler?
	// slog logic: Handle gets Record. Record has attributes.
	// So we can wrap the *next* handler?
	// No, Record is immutable value.
	// For Async, simple approach: just pass through.
	// But Wait, WithAttrs returns a *new Handler*.
	return &AsyncHandler{
		next:       h.next.WithAttrs(attrs),
		buffer:     h.buffer, // Share buffer? Yes
		done:       h.done,
		dropOnFull: h.dropOnFull,
	}
}

func (h *AsyncHandler) WithGroup(name string) slog.Handler {
	return &AsyncHandler{
		next:       h.next.WithGroup(name),
		buffer:     h.buffer,
		done:       h.done,
		dropOnFull: h.dropOnFull,
	}
}

func (h *AsyncHandler) Shutdown() {
	close(h.buffer)
	<-h.done
}

// --- Redact Handler ---

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
var creditCardRegex = regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`)

type RedactHandler struct {
	next slog.Handler
}

func NewRedactHandler(next slog.Handler) *RedactHandler {
	return &RedactHandler{next: next}
}

func (h *RedactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *RedactHandler) Handle(ctx context.Context, r slog.Record) error {
	// We must walk all attributes and redact strings
	newAttrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		newAttrs = append(newAttrs, h.redactAttr(a))
		return true
	})

	// Create new record with redacted attributes
	r2 := slog.NewRecord(r.Time, r.Level, h.redactString(r.Message), r.PC)
	r2.AddAttrs(newAttrs...)

	return h.next.Handle(ctx, r2)
}

func (h *RedactHandler) redactAttr(a slog.Attr) slog.Attr {
	// Recursive for groups
	if a.Value.Kind() == slog.KindGroup {
		groupAttrs := a.Value.Group()
		newGroup := make([]slog.Attr, len(groupAttrs))
		for i, sub := range groupAttrs {
			newGroup[i] = h.redactAttr(sub)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(newGroup...)}
	}

	if a.Value.Kind() == slog.KindString {
		// Identify specific keys?
		key := strings.ToLower(a.Key)
		if strings.Contains(key, "token") || strings.Contains(key, "password") || strings.Contains(key, "secret") ||
			strings.Contains(key, "api_key") || strings.Contains(key, "apikey") || strings.Contains(key, "access_key") ||
			strings.Contains(key, "authorization") || strings.Contains(key, "cookie") || strings.Contains(key, "bearer") {
			return slog.String(a.Key, "[REDACTED]")
		}

		// Regex scan value
		return slog.String(a.Key, h.redactString(a.Value.String()))
	}
	return a
}

func (h *RedactHandler) redactString(s string) string {
	s = emailRegex.ReplaceAllString(s, "[EMAIL]")
	s = creditCardRegex.ReplaceAllString(s, "[CC]")
	return s
}

func (h *RedactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &RedactHandler{next: h.next.WithAttrs(attrs)}
}

func (h *RedactHandler) WithGroup(name string) slog.Handler {
	return &RedactHandler{next: h.next.WithGroup(name)}
}

// --- Tee Handler ---

// TeeHandler duplicates logs to multiple handlers.
type TeeHandler struct {
	handlers []slog.Handler
}

func NewTeeHandler(handlers ...slog.Handler) *TeeHandler {
	return &TeeHandler{handlers: handlers}
}

func (h *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			_ = handler.Handle(ctx, r)
		}
	}
	return nil
}

func (h *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		next[i] = handler.WithAttrs(attrs)
	}
	return NewTeeHandler(next...)
}

func (h *TeeHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		next[i] = handler.WithGroup(name)
	}
	return NewTeeHandler(next...)
}
