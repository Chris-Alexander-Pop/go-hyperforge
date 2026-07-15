package sql

import (
	"context"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/sharding"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"gorm.io/gorm"
)

// Sharded routes GetShard calls across multiple SQL backends using a
// sharding.Strategy (typically consistent hashing). Single-instance adapters
// ignore the shard key; use this type when you need real multi-shard routing.
type Sharded struct {
	strategy sharding.Strategy
	shards   map[string]SQL
	primary  string
	mu       sync.RWMutex
}

// Ensure Sharded implements SQL.
var _ SQL = (*Sharded)(nil)

// NewSharded creates a multi-shard SQL helper.
// shards maps shard IDs (as returned by strategy.GetShard) to SQL backends.
// primary is the shard used for Get(); if empty, an arbitrary registered shard is chosen.
func NewSharded(strategy sharding.Strategy, shards map[string]SQL, primary string) (*Sharded, error) {
	if strategy == nil {
		return nil, errors.InvalidArgument("sharding strategy is required", nil)
	}
	if len(shards) == 0 {
		return nil, errors.InvalidArgument("at least one shard is required", nil)
	}

	if primary == "" {
		for id := range shards {
			primary = id
			break
		}
	}
	if _, ok := shards[primary]; !ok {
		return nil, errors.InvalidArgument("primary shard not found in shards map", nil)
	}

	// Copy to avoid caller mutation.
	copied := make(map[string]SQL, len(shards))
	for id, db := range shards {
		if db == nil {
			return nil, errors.InvalidArgument("nil shard: "+id, nil)
		}
		copied[id] = db
	}

	return &Sharded{
		strategy: strategy,
		shards:   copied,
		primary:  primary,
	}, nil
}

// Get returns the primary shard connection.
func (s *Sharded) Get(ctx context.Context) *gorm.DB {
	s.mu.RLock()
	db := s.shards[s.primary]
	s.mu.RUnlock()
	return db.Get(ctx)
}

// GetShard resolves the shard for key via the strategy and returns that backend's connection.
func (s *Sharded) GetShard(ctx context.Context, key string) (*gorm.DB, error) {
	id := s.strategy.GetShard(key)
	if id == "" {
		return nil, ErrShardNotFound
	}

	s.mu.RLock()
	db, ok := s.shards[id]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrShardNotFound
	}
	return db.Get(ctx), nil
}

// Close closes all registered shard backends. Returns the first error encountered.
func (s *Sharded) Close() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var first error
	for _, db := range s.shards {
		if err := db.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// ShardIDs returns the registered shard identifiers.
func (s *Sharded) ShardIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.shards))
	for id := range s.shards {
		ids = append(ids, id)
	}
	return ids
}
