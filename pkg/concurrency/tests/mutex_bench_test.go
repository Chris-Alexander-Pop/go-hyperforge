package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// BenchmarkRawRWMutex benchmarks the standard library sync.RWMutex
func BenchmarkRawRWMutex_Lock(b *testing.B) {
	var mu sync.RWMutex
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		mu.Unlock()
	}
}

func BenchmarkRawRWMutex_RLock(b *testing.B) {
	var mu sync.RWMutex
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		mu.RUnlock()
	}
}

// BenchmarkSmartRWMutex_Default benchmarks default mode (fast, no observability)
func BenchmarkSmartRWMutex_Default_Lock(b *testing.B) {
	mu := concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "bench"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		mu.Unlock()
	}
}

// BenchmarkSmartRWMutex_Default benchmarks default mode (fast, no observability)
func BenchmarkSmartRWMutex_Default_RLock(b *testing.B) {
	mu := concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "bench"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		mu.RUnlock()
	}
}

// Benchmark the overhead of SmartRWMutex vs sync.RWMutex in various scenarios.

func BenchmarkNativeRWMutex_LockUnlock(b *testing.B) {
	var mu sync.RWMutex
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			mu.Unlock()
		}
	})
}

func BenchmarkSmartRWMutex_LockUnlock_FastPath(b *testing.B) {
	mu := concurrency.NewSmartRWMutex(concurrency.MutexConfig{
		Name:      "bench-fast",
		DebugMode: false,
	})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			mu.Unlock()
		}
	})
}

func BenchmarkSmartRWMutex_LockUnlock_DebugMode(b *testing.B) {
	mu := concurrency.NewSmartRWMutex(concurrency.MutexConfig{
		Name:      "bench-debug",
		DebugMode: true,
	})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			mu.Unlock()
		}
	})
}

func BenchmarkNativeRWMutex_RLockRUnlock(b *testing.B) {
	var mu sync.RWMutex
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			mu.RUnlock()
		}
	})
}

func BenchmarkSmartRWMutex_RLockRUnlock_FastPath(b *testing.B) {
	mu := concurrency.NewSmartRWMutex(concurrency.MutexConfig{
		Name:      "bench-fast",
		DebugMode: false,
	})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			mu.RUnlock()
		}
	})
}

func BenchmarkSmartRWMutex_RLockRUnlock_DebugMode(b *testing.B) {
	mu := concurrency.NewSmartRWMutex(concurrency.MutexConfig{
		Name:      "bench-debug",
		DebugMode: true,
	})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			mu.RUnlock()
		}
	})
}

// Contention benchmarks
func BenchmarkSmartRWMutex_HighContention_Debug(b *testing.B) {
	mu := concurrency.NewSmartRWMutex(concurrency.MutexConfig{DebugMode: true})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			time.Sleep(100 * time.Nanosecond)
			mu.Unlock()
		}
	})
}

func BenchmarkNativeRWMutex_HighContention(b *testing.B) {
	var mu sync.RWMutex
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			time.Sleep(100 * time.Nanosecond)
			mu.Unlock()
		}
	})
}
