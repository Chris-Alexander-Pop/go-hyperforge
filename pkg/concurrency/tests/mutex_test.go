package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

func TestSmartMutex(t *testing.T) {
	mu := concurrency.NewSmartMutex(concurrency.MutexConfig{
		Name:      "test-mutex",
		DebugMode: true,
	})

	// Basic Lock/Unlock
	mu.Lock()
	mu.Unlock()

	// Concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
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
	mu.Unlock()

	// Read Lock
	mu.RLock()
	mu.RUnlock()

	// Concurrent Reads
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.RLock()
			time.Sleep(1 * time.Millisecond)
			mu.RUnlock()
		}()
	}
	wg.Wait()
}
