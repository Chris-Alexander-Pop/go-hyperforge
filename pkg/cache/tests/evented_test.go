package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
)

func TestEventedCache_SetDelete(t *testing.T) {
	bus := eventsmem.New(events.Config{})
	t.Cleanup(func() { _ = bus.Close() })

	var got events.Event
	done := make(chan struct{}, 1)
	_, err := bus.Subscribe(context.Background(), cache.TopicCache, func(ctx context.Context, ev events.Event) error {
		got = ev
		done <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	c := cache.NewEventedCache(memory.New(), bus)
	t.Cleanup(func() { _ = c.Close() })

	if err := c.Set(context.Background(), "k", "v", time.Minute); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout set")
	}
	if got.Type != cache.EventTypeSet {
		t.Fatalf("type=%s", got.Type)
	}

	if err := c.Delete(context.Background(), "k"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout delete")
	}
	if got.Type != cache.EventTypeDeleted {
		t.Fatalf("type=%s", got.Type)
	}
}

func TestEventedCache_NilBus(t *testing.T) {
	c := cache.NewEventedCache(memory.New(), nil)
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Set(context.Background(), "k", "v", 0); err != nil {
		t.Fatal(err)
	}
}
