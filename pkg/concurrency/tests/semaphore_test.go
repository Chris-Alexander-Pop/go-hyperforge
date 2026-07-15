package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

func TestSemaphoreAcquireCancel(t *testing.T) {
	sem := concurrency.NewSemaphore(1)

	if err := sem.Acquire(context.Background(), 1); err != nil {
		t.Fatalf("initial acquire: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- sem.Acquire(ctx, 1)
	}()

	// Give the waiter time to enqueue.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected context cancellation error")
		}
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Acquire did not return after cancel")
	}

	// Slot still held by first acquire; release and ensure a new acquire works.
	sem.Release(1)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if err := sem.Acquire(ctx2, 1); err != nil {
		t.Fatalf("acquire after cancel cleanup: %v", err)
	}
	sem.Release(1)
}

func TestSemaphoreAcquireCancelWhileNotified(t *testing.T) {
	sem := concurrency.NewSemaphore(1)
	if err := sem.Acquire(context.Background(), 1); err != nil {
		t.Fatalf("initial acquire: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	errCh := make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer wg.Done()
		errCh <- sem.Acquire(ctx, 1)
	}()

	time.Sleep(20 * time.Millisecond)

	// Release so the waiter may be notified; cancel concurrently.
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	sem.Release(1)

	wg.Wait()
	err := <-errCh
	// Either we acquired successfully before cancel, or we got canceled.
	// Both are acceptable; the important part is no hang / no panic / no leak.
	if err != nil && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
	if err == nil {
		sem.Release(1)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if err := sem.Acquire(ctx2, 1); err != nil {
		t.Fatalf("semaphore stuck after cancel race: %v", err)
	}
	sem.Release(1)
}

func TestSemaphoreTryAcquire(t *testing.T) {
	sem := concurrency.NewSemaphore(1)
	if !sem.TryAcquire(1) {
		t.Fatal("expected TryAcquire to succeed")
	}
	if sem.TryAcquire(1) {
		t.Fatal("expected TryAcquire to fail when exhausted")
	}
	sem.Release(1)
	if !sem.TryAcquire(1) {
		t.Fatal("expected TryAcquire to succeed after release")
	}
	sem.Release(1)
}
