package cache

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/bloomfilter"
)

// BloomCache wraps a Cache with a Bloom filter for negative caching.
// This prevents expensive cache/database lookups for keys that definitely don't exist.
//
// Use case: If your cache backs a database, a cache miss triggers a DB query.
// With BloomCache, we first check the Bloom filter - if it says "no", we skip
// the lookup entirely, saving latency and load.
type BloomCache struct {
	cache  Cache
	bloom  *bloomfilter.BloomFilter
	prefix string
}

// BloomCacheConfig configures the Bloom filter cache.
type BloomCacheConfig struct {
	// ExpectedElements is the estimated number of unique keys.
	ExpectedElements uint `env:"CACHE_BLOOM_ELEMENTS" env-default:"100000"`

	// FalsePositiveRate is the acceptable false positive rate (0.01 = 1%).
	FalsePositiveRate float64 `env:"CACHE_BLOOM_FPR" env-default:"0.01"`

	// Prefix is added to keys for namespacing.
	Prefix string `env:"CACHE_BLOOM_PREFIX" env-default:""`
}

// NewBloomCache wraps a cache with a Bloom filter for negative lookups.
func NewBloomCache(cache Cache, cfg BloomCacheConfig) *BloomCache {
	return &BloomCache{
		cache:  cache,
		bloom:  bloomfilter.New(cfg.ExpectedElements, cfg.FalsePositiveRate),
		prefix: cfg.Prefix,
	}
}

func (bc *BloomCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := bc.prefix + key

	// Fast path: if Bloom filter says "no", definitely not in cache
	if !bc.bloom.ContainsString(fullKey) {
		return ErrKeyNotFound
	}

	// Bloom filter says "maybe" - check actual cache
	return bc.cache.Get(ctx, key, dest)
}

func (bc *BloomCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := bc.prefix + key
	bc.bloom.AddString(fullKey)
	return bc.cache.Set(ctx, key, value, ttl)
}

func (bc *BloomCache) Delete(ctx context.Context, key string) error {
	// Note: Bloom filters don't support deletion.
	// The key will remain in the filter (false positive), but that's acceptable.
	return bc.cache.Delete(ctx, key)
}

func (bc *BloomCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := bc.prefix + key
	if !bc.bloom.ContainsString(fullKey) {
		return false, nil
	}
	return bc.cache.Exists(ctx, key)
}

func (bc *BloomCache) MGet(ctx context.Context, keys []string, dest interface{}) error {
	filtered := make([]string, 0, len(keys))
	for _, key := range keys {
		if bc.bloom.ContainsString(bc.prefix + key) {
			filtered = append(filtered, key)
		}
	}
	return bc.cache.MGet(ctx, filtered, dest)
}

func (bc *BloomCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key := range items {
		bc.bloom.AddString(bc.prefix + key)
	}
	return bc.cache.MSet(ctx, items, ttl)
}

func (bc *BloomCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return bc.cache.Expire(ctx, key, ttl)
}

func (bc *BloomCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return bc.cache.GetTTL(ctx, key)
}

func (bc *BloomCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := bc.prefix + key
	bc.bloom.AddString(fullKey)
	return bc.cache.Incr(ctx, key, delta)
}

func (bc *BloomCache) Close() error {
	return bc.cache.Close()
}

// Unwrap returns the underlying cache.
func (bc *BloomCache) Unwrap() Cache {
	return bc.cache
}

// Stats returns Bloom filter statistics.
func (bc *BloomCache) Stats() BloomCacheStats {
	return BloomCacheStats{
		Elements:          bc.bloom.Count(),
		FalsePositiveRate: bc.bloom.EstimatedFalsePositiveRate(),
	}
}

// BloomCacheStats contains Bloom filter statistics.
type BloomCacheStats struct {
	Elements          uint64
	FalsePositiveRate float64
}

// Warm pre-populates the Bloom filter with existing keys.
// Call this on startup if you have a list of existing keys.
func (bc *BloomCache) Warm(keys []string) {
	for _, key := range keys {
		bc.bloom.AddString(bc.prefix + key)
	}
}
