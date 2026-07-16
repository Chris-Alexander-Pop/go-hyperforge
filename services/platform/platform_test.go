package platform_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"github.com/chris-alexander-pop/go-hyperforge/services/platform"
)

func TestLoadAndInitLogger(t *testing.T) {
	t.Setenv("SERVICE_NAME", "test-svc")
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")

	var cfg platform.BaseConfig
	if err := platform.Load(&cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	platform.InitLogger(cfg.LogLevel)

	if cfg.ServiceName != "test-svc" {
		t.Fatalf("ServiceName = %q", cfg.ServiceName)
	}
	if cfg.Port != "9090" {
		t.Fatalf("Port = %q", cfg.Port)
	}
	_ = logger.Shutdown(context.Background())
}

func TestWaitForShutdown(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		done <- platform.WaitForShutdown(time.Second, func(ctx context.Context) error {
			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess: %v", err)
	}
	if err := p.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WaitForShutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for shutdown")
	}
}
