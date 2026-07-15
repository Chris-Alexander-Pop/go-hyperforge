package htb

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"time"
)

// Bucket represents a node in the HTB hierarchy.
type Bucket struct {
	ID         string
	Parent     *Bucket
	Rate       float64 // tokens per second
	Capacity   float64
	Tokens     float64
	LastRefill time.Time
	Children   []*Bucket
	mu         *concurrency.SmartMutex
}

// New creates a new bucket.
func New(id string, rate, capacity float64) *Bucket {
	return &Bucket{
		ID:         id,
		Rate:       rate,
		Capacity:   capacity,
		Tokens:     capacity, // Full start
		LastRefill: time.Now(),
		mu:         concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "htb-bucket"}),
	}
}

// AddChild attaches a bucket as a child.
func (b *Bucket) AddChild(child *Bucket) {
	b.mu.Lock()
	defer b.mu.Unlock()
	child.Parent = b // Assume single parent for tree structure
	b.Children = append(b.Children, child)
}

// Allow limits requests. It must satisfy this bucket AND all ancestors.
// This supports "Global API Limit" > "Tenant Limit" > "User Limit".
func (b *Bucket) Allow(cost float64) bool {
	// Need to lock path to root or use atomic subtractions with fallback.
	// Locking path is safe but contention heavy.
	// Let's assume low depth and lock from root down? or just verify up?

	// Verified approach: Check local, then recursively check parent.
	// Issue: If parent succeeds but child fails? Or vice versa?
	// Transactional nature required.

	// Simplest: Global lock (bad).
	// Medium: Lock from parent to child (deadlock avoidance if tree structure is strict).

	// Let's implement recursive consumption.
	// If Allow() is called, we must consume tokens from ALL buckets in chain.

	path := b.getPath()

	// Lock all
	for _, bucket := range path {
		bucket.mu.Lock()
	}
	defer func() {
		for _, bucket := range path {
			bucket.mu.Unlock()
		}
	}()

	now := time.Now()

	// 1. Refill all
	for _, bucket := range path {
		elapsed := now.Sub(bucket.LastRefill).Seconds()
		refill := elapsed * bucket.Rate
		bucket.Tokens += refill
		if bucket.Tokens > bucket.Capacity {
			bucket.Tokens = bucket.Capacity
		}
		bucket.LastRefill = now
	}

	// 2. Check check all have enough
	for _, bucket := range path {
		if bucket.Tokens < cost {
			return false
		}
	}

	// 3. Consume
	for _, bucket := range path {
		bucket.Tokens -= cost
	}

	return true
}

func (b *Bucket) getPath() []*Bucket {
	var path []*Bucket
	curr := b
	for curr != nil {
		path = append(path, curr) // Child first
		curr = curr.Parent
	}
	// Reverse to get Root -> Child (Leaf).
	// Actually order for locking assumes Root -> Leaf for strict ordering if multi-access.
	// Or Leaf -> Root?
	// If traversing down, Root first.
	// So reverse list.

	n := len(path)
	reversed := make([]*Bucket, n)
	for i := 0; i < n; i++ {
		reversed[i] = path[n-1-i]
	}
	return reversed
}
