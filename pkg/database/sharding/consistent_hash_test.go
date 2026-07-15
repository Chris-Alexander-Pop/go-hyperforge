package sharding_test

import (
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/sharding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsistentHash_GetShard_Deterministic(t *testing.T) {
	shards := []string{"shard-a", "shard-b", "shard-c"}
	ch := sharding.NewConsistentHash(50, shards)

	key := "user:42"
	first := ch.GetShard(key)
	require.NotEmpty(t, first)
	assert.Contains(t, shards, first)

	for i := 0; i < 20; i++ {
		assert.Equal(t, first, ch.GetShard(key), "same key must map to same shard")
	}
}

func TestConsistentHash_Distribution(t *testing.T) {
	shards := []string{"s0", "s1", "s2", "s3"}
	ch := sharding.NewConsistentHash(100, shards)

	counts := map[string]int{}
	for i := 0; i < 400; i++ {
		id := ch.GetShard(fmt.Sprintf("key-%d", i))
		require.Contains(t, shards, id)
		counts[id]++
	}

	for _, s := range shards {
		assert.Greater(t, counts[s], 0, "shard %s got no keys", s)
	}
}

func TestConsistentHash_AddRemove(t *testing.T) {
	ch := sharding.NewConsistentHash(50, []string{"a", "b"})
	before := ch.GetShard("stable-key")
	require.NotEmpty(t, before)

	ch.AddShard("c")
	afterAdd := ch.GetShard("stable-key")
	assert.Contains(t, []string{"a", "b", "c"}, afterAdd)

	ch.RemoveShard("c")
	afterRemove := ch.GetShard("stable-key")
	assert.Contains(t, []string{"a", "b"}, afterRemove)
}

func TestConsistentHash_EmptyRing(t *testing.T) {
	ch := sharding.NewConsistentHash(10, nil)
	assert.Equal(t, "", ch.GetShard("anything"))
}

func TestConsistentHash_ImplementsStrategy(t *testing.T) {
	var _ sharding.Strategy = sharding.NewConsistentHash(10, []string{"x"})
}
