package cache_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
)

func BenchmarkMemoryCache_Set(b *testing.B) {
	c := memory.New()
	defer c.Close()

	ctx := context.Background()
	value := map[string]string{"name": "test", "email": "test@example.com"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = c.Set(ctx, key, value, time.Hour)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	c := memory.New()
	defer c.Close()

	ctx := context.Background()
	value := map[string]string{"name": "test", "email": "test@example.com"}

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = c.Set(ctx, key, value, time.Hour)
	}

	b.ResetTimer()
	var result map[string]string
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i%1000)
		_ = c.Get(ctx, key, &result)
	}
}

func BenchmarkMemoryCache_SetGet(b *testing.B) {
	c := memory.New()
	defer c.Close()

	ctx := context.Background()
	value := "benchmark-value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = c.Set(ctx, key, value, time.Hour)

		var result string
		_ = c.Get(ctx, key, &result)
	}
}

func BenchmarkMemoryCache_Parallel(b *testing.B) {
	c := memory.New()
	defer c.Close()

	ctx := context.Background()
	value := "parallel-test"

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("pkey:%d", i)
			_ = c.Set(ctx, key, value, time.Hour)

			var result string
			_ = c.Get(ctx, key, &result)
			i++
		}
	})
}
