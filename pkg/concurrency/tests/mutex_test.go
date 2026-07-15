package tests

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// testCounter is used to prevent empty critical section warnings.
var testCounter atomic.Int64

func TestSmartMutex(t *testing.T) {
	mu := concurrency.NewSmartMutex(concurrency.MutexConfig{
		Name:      "test-mutex",
		DebugMode: true,
	})

	// Basic Lock/Unlock
	mu.Lock()
	testCounter.Add(1)
	mu.Unlock()

	// Concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			testCounter.Add(1)
			time.Sleep(1 * time.Millisecond)
			mu.Unlock()
		}()
	}
	wg.Wait()
}

func TestSmartRWMutex(t *testing.T) {
	mu := concurrency.NewSmartRWMutex(concurrency.MutexConfig{
		Name:      "test-rwmutex",
		DebugMode: true,
	})

	// Write Lock
	mu.Lock()
	testCounter.Add(1)
	mu.Unlock()

	// Read Lock
	mu.RLock()
	_ = testCounter.Load()
	mu.RUnlock()

	// Concurrent Reads
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.RLock()
			_ = testCounter.Load()
			time.Sleep(1 * time.Millisecond)
			mu.RUnlock()
		}()
	}
	wg.Wait()
}
