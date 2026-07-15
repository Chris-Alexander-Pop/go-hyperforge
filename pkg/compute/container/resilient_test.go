package container_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/compute/container"
	"github.com/chris-alexander-pop/system-design-library/pkg/compute/container/adapters/memory"
)

func TestResilientRuntimePassesThrough(t *testing.T) {
	inner := memory.New()
	rt := container.NewResilientRuntime(inner, container.ResilientConfig{
		RetryEnabled:     true,
		RetryMaxAttempts: 2,
		RetryBackoff:     time.Millisecond,
	})

	ctx := context.Background()
	c, err := rt.Create(ctx, container.CreateOptions{Image: "nginx", Name: "web"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := rt.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != c.ID {
		t.Fatalf("id mismatch")
	}
	if err := rt.Start(ctx, c.ID); err != nil {
		t.Fatalf("Start: %v", err)
	}
}

func TestResilientRuntimeDoesNotRetryNotFound(t *testing.T) {
	inner := &countingRuntime{Runtime: memory.New()}
	rt := container.NewResilientRuntime(inner, container.ResilientConfig{
		RetryEnabled:     true,
		RetryMaxAttempts: 5,
		RetryBackoff:     time.Millisecond,
	})

	_, err := rt.Get(context.Background(), "missing")
	if err != container.ErrContainerNotFound {
		t.Fatalf("expected ErrContainerNotFound, got %v", err)
	}
	if inner.getCalls != 1 {
		t.Fatalf("expected single Get attempt for NotFound, got %d", inner.getCalls)
	}
}

type countingRuntime struct {
	*memory.Runtime
	getCalls int
}

func (c *countingRuntime) Get(ctx context.Context, id string) (*container.Container, error) {
	c.getCalls++
	return c.Runtime.Get(ctx, id)
}

var _ container.ContainerRuntime = (*countingRuntime)(nil)
