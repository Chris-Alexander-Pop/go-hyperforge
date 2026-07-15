package blob_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/adapters/memory"
)

func TestResilientStore_DownloadMissNotRetried(t *testing.T) {
	inner := memory.New(blob.Config{})
	store := blob.NewResilientStore(inner, blob.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      3,
		RetryBackoff:          time.Millisecond,
	})

	_, err := store.Download(context.Background(), "missing")
	if !blob.IsNotFound(err) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestEventedStore_TypedPayload(t *testing.T) {
	inner := memory.New(blob.Config{})
	bus := eventsmemory.New(events.DefaultConfig())
	t.Cleanup(func() { _ = bus.Close() })

	var got events.Event
	done := make(chan struct{}, 1)
	_, err := bus.Subscribe(context.Background(), blob.TopicBlob, func(ctx context.Context, event events.Event) error {
		got = event
		done <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	store := blob.NewEventedStore(inner, bus)
	if err := store.Upload(context.Background(), "k1", strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	if got.Type != blob.EventTypeUploaded {
		t.Fatalf("type=%s", got.Type)
	}
	payload, ok := got.Payload.(blob.BlobEventPayload)
	if !ok {
		t.Fatalf("payload type %T, want BlobEventPayload", got.Payload)
	}
	if payload.Key != "k1" {
		t.Fatalf("key=%s", payload.Key)
	}
}

func TestEventedStore_NilBus(t *testing.T) {
	inner := memory.New(blob.Config{})
	store := blob.NewEventedStore(inner, nil)
	if err := store.Upload(context.Background(), "k", strings.NewReader("x")); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryStore_UploadDownloadDelete(t *testing.T) {
	store := memory.New(blob.Config{})
	ctx := context.Background()

	if err := store.Upload(ctx, "a/b", strings.NewReader("data")); err != nil {
		t.Fatal(err)
	}
	rc, err := store.Download(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, []byte("data")) {
		t.Fatalf("body=%q", body)
	}
	if err := store.Delete(ctx, "a/b"); err != nil {
		t.Fatal(err)
	}
	_, err = store.Download(ctx, "a/b")
	if !blob.IsNotFound(err) {
		t.Fatalf("expected NotFound, got %v", err)
	}
	if !errors.Is(err, blob.ErrNotFound) {
		t.Fatalf("expected ErrNotFound sentinel, got %v", err)
	}
}
