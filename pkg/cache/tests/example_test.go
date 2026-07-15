package cache_test

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
)

func Example() {
	// Create a memory cache
	c := memory.New()
	defer c.Close()

	ctx := context.Background()

	// Store a value
	err := c.Set(ctx, "user:123", map[string]string{"name": "Alice"}, time.Hour)
	if err != nil {
		panic(err)
	}

	// Retrieve the value
	var user map[string]string
	err = c.Get(ctx, "user:123", &user)
	if err != nil {
		panic(err)
	}

	fmt.Println(user["name"])
	// Output: Alice
}

func ExampleCache_withInstrumentation() {
	// Create a memory cache with instrumentation
	memCache := memory.New()
	c := cache.NewInstrumentedCache(memCache)
	defer c.Close()

	ctx := context.Background()

	// All operations are now traced and logged
	_ = c.Set(ctx, "key", "value", time.Minute)

	var result string
	_ = c.Get(ctx, "key", &result)

	fmt.Println(result)
	// Output: value
}
