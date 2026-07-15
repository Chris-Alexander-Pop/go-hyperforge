package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

func TestGroupDoCoalesces(t *testing.T) {
	var g concurrency.Group
	var calls atomic.Int32

	var wg sync.WaitGroup
	const n = 32
	results := make([]interface{}, n)
	sharedFlags := make([]bool, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, err, shared := g.Do("key", func() (interface{}, error) {
				calls.Add(1)
				time.Sleep(20 * time.Millisecond)
				return "ok", nil
			})
			if err != nil {
				t.Errorf("Do: %v", err)
			}
			results[i] = v
			sharedFlags[i] = shared
		}(i)
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 call, got %d", got)
	}
	sharedCount := 0
	for i := 0; i < n; i++ {
		if results[i] != "ok" {
			t.Fatalf("result[%d]=%v", i, results[i])
		}
		if sharedFlags[i] {
			sharedCount++
		}
	}
	if sharedCount < 1 {
		t.Fatal("expected some callers to see shared=true")
	}
}

func TestGroupForget(t *testing.T) {
	var g concurrency.Group
	started := make(chan struct{})
	release := make(chan struct{})

	go func() {
		_, _, _ = g.Do("k", func() (interface{}, error) {
			close(started)
			<-release
			return 1, nil
		})
	}()
	<-started
	g.Forget("k")

	v, err, shared := g.Do("k", func() (interface{}, error) {
		return 2, nil
	})
	if err != nil || shared || v != 2 {
		t.Fatalf("after Forget: v=%v err=%v shared=%v", v, err, shared)
	}
	close(release)
}

func TestWorkerPoolAdaptive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp := concurrency.NewWorkerPool(2, 8, concurrency.WithAdaptiveLimiter(1, 4))
	wp.Start(ctx)

	var ran atomic.Int32
	for i := 0; i < 5; i++ {
		wp.Submit(func(ctx context.Context) {
			ran.Add(1)
			time.Sleep(5 * time.Millisecond)
		})
	}
	time.Sleep(100 * time.Millisecond)
	cancel()
	wp.Stop()

	if ran.Load() == 0 {
		t.Fatal("expected adaptive pool to run tasks")
	}
	if lim := wp.AdaptiveLimit(); lim < 1 {
		t.Fatalf("adaptive limit=%v", lim)
	}
}
