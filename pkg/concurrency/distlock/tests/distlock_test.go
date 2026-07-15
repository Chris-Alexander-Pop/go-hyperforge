package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency/distlock"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency/distlock/adapters/memory"
)

func TestDistLock(t *testing.T) {
	locker := memory.New()
	defer locker.Close()

	ctx := context.Background()
	key := "resource-1"

	// 1. Acquire Lock
	lock1 := locker.NewLock(key, time.Second)
	acquired, err := lock1.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if !acquired {
		t.Fatal("Expected to acquire lock")
	}

	// 2. Try Acquire same lock (should fail)
	lock2 := locker.NewLock(key, time.Second)
	acquired2, err := lock2.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire 2 failed: %v", err)
	}
	if acquired2 {
		t.Fatal("Expected NOT to acquire lock 2")
	}

	// 3. Release
	err = lock1.Release(ctx)
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// 4. Acquire again (should succeed)
	acquired3, err := lock2.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire 3 failed: %v", err)
	}
	if !acquired3 {
		t.Fatal("Expected to acquire lock 2 after release")
	}
}

func TestAcquireWithRetry(t *testing.T) {
	locker := memory.New()
	defer locker.Close()

	ctx := context.Background()
	key := "retry-resource"

	holder := locker.NewLock(key, 200*time.Millisecond)
	ok, err := holder.Acquire(ctx)
	if err != nil || !ok {
		t.Fatalf("holder acquire: ok=%v err=%v", ok, err)
	}

	waiter := locker.NewLock(key, time.Second)
	cfg := distlock.LockConfig{
		TTL:        time.Second,
		RetryDelay: 20 * time.Millisecond,
		RetryCount: 20,
	}

	start := time.Now()
	acquired, err := distlock.AcquireWithRetry(ctx, waiter, cfg)
	if err != nil {
		t.Fatalf("AcquireWithRetry: %v", err)
	}
	if !acquired {
		t.Fatal("expected AcquireWithRetry to succeed after TTL expiry")
	}
	if time.Since(start) < 150*time.Millisecond {
		t.Fatalf("expected retries while lock held, elapsed=%v", time.Since(start))
	}
}

func TestAcquireWithRetryCancel(t *testing.T) {
	locker := memory.New()
	defer locker.Close()

	holder := locker.NewLock("cancel-key", time.Minute)
	ok, err := holder.Acquire(context.Background())
	if err != nil || !ok {
		t.Fatalf("holder acquire: ok=%v err=%v", ok, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	waiter := locker.NewLock("cancel-key", time.Second)
	acquired, err := distlock.AcquireWithRetry(ctx, waiter, distlock.LockConfig{
		RetryDelay: 10 * time.Millisecond,
		RetryCount: 50,
	})
	if acquired {
		t.Fatal("should not acquire while held")
	}
	if err == nil {
		t.Fatal("expected context error")
	}
}
