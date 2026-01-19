package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency/distlock/adapters/memory"
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
