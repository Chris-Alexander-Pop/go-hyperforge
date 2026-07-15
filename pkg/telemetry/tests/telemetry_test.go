package telemetry_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestInitNoop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	shutdown, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName: "test-noop",
		Provider:    telemetry.ProviderNoop,
		SampleRate:  1.0,
	})
	if err != nil {
		t.Fatalf("Init noop: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown")
	}

	tr := otel.Tracer("telemetry_test")
	_, span := tr.Start(ctx, "noop.span")
	span.End()

	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestInitStdout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var buf bytes.Buffer
	shutdown, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:  "test-stdout",
		Provider:     telemetry.ProviderStdout,
		SampleRate:   1.0,
		StdoutWriter: &buf,
	})
	if err != nil {
		t.Fatalf("Init stdout: %v", err)
	}

	tr := otel.Tracer("telemetry_test")
	_, span := tr.Start(ctx, "stdout.span")
	span.End()

	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected stdout exporter to write span data")
	}
}

func TestInitSampleRateZero(t *testing.T) {
	ctx := context.Background()
	shutdown, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName: "test-sample-zero",
		Provider:    telemetry.ProviderNoop,
		SampleRate:  0,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer shutdown(ctx)

	tr := otel.Tracer("telemetry_test")
	_, span := tr.Start(ctx, "never.sampled")
	if span.SpanContext().IsSampled() {
		t.Fatal("expected NeverSample when SampleRate=0")
	}
	span.End()
}

func TestRecordErrorAndSetStatus(t *testing.T) {
	ctx := context.Background()
	shutdown, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName: "test-helpers",
		Provider:    telemetry.ProviderNoop,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer shutdown(ctx)

	tr := telemetry.Tracer("telemetry_test")
	_, span := tr.Start(ctx, "helper.span")

	telemetry.RecordError(span, nil) // no-op
	telemetry.RecordError(span, errors.New("boom"))
	telemetry.SetStatus(span, codes.Ok, "")
	span.End()
}

func TestInitDoesNotHangWithoutCollector(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		shutdown, err := telemetry.Init(ctx, telemetry.Config{
			ServiceName: "test-fast",
			Provider:    telemetry.ProviderNoop,
		})
		if err != nil {
			done <- err
			return
		}
		done <- shutdown(context.Background())
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("noop Init/shutdown failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Init hung — noop path must not contact a collector")
	}
}

func TestOTLPConfigFields(t *testing.T) {
	cfg := telemetry.Config{
		ServiceName:    "svc",
		ServiceVersion: "1.2.3",
		Environment:    "test",
		Endpoint:       "collector:4317",
		Provider:       telemetry.ProviderOTLP,
		SampleRate:     0.5,
		Insecure:       true,
	}
	if cfg.SampleRate != 0.5 {
		t.Fatalf("SampleRate=%v", cfg.SampleRate)
	}
	if !cfg.Insecure {
		t.Fatal("expected Insecure true")
	}
	if !strings.EqualFold(cfg.Provider, telemetry.ProviderOTLP) {
		t.Fatalf("Provider=%s", cfg.Provider)
	}
	_ = trace.SpanFromContext(context.Background())
}
