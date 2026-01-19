package concurrency

import (
	"sync"
	"testing"
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
	mu := NewSmartRWMutex(MutexConfig{Name: "bench"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		mu.Unlock()
	}
}

func BenchmarkSmartRWMutex_Default_RLock(b *testing.B) {
	mu := NewSmartRWMutex(MutexConfig{Name: "bench"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		mu.RUnlock()
	}
}

// BenchmarkSmartRWMutex_DebugMode benchmarks with observability enabled
func BenchmarkSmartRWMutex_DebugMode_Lock(b *testing.B) {
	mu := NewSmartRWMutex(MutexConfig{Name: "bench", DebugMode: true})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		mu.Unlock()
	}
}

func BenchmarkSmartRWMutex_DebugMode_RLock(b *testing.B) {
	mu := NewSmartRWMutex(MutexConfig{Name: "bench", DebugMode: true})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		mu.RUnlock()
	}
}

// Parallel benchmarks (more realistic high-contention scenario)
func BenchmarkRawRWMutex_Parallel_RLock(b *testing.B) {
	var mu sync.RWMutex
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			mu.RUnlock()
		}
	})
}

func BenchmarkSmartRWMutex_Default_Parallel_RLock(b *testing.B) {
	mu := NewSmartRWMutex(MutexConfig{Name: "bench"})
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			mu.RUnlock()
		}
	})
}

func BenchmarkSmartRWMutex_DebugMode_Parallel_RLock(b *testing.B) {
	mu := NewSmartRWMutex(MutexConfig{Name: "bench", DebugMode: true})
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			mu.RUnlock()
		}
	})
}
