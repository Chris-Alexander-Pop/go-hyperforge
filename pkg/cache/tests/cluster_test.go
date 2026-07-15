package cache

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	cacheredis "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/redis"
)

func TestClusterConfigRequiresSeed(t *testing.T) {
	_, err := cacheredis.New(cache.Config{Cluster: true})
	if err == nil {
		t.Fatal("expected error when Cluster=true without Addrs/Host")
	}
}

func TestClusterConfigFieldPresent(t *testing.T) {
	cfg := cache.Config{
		Cluster: true,
		Addrs:   []string{"127.0.0.1:7000", "127.0.0.1:7001"},
	}
	if !cfg.Cluster || len(cfg.Addrs) != 2 {
		t.Fatalf("ClusterConfig fields not set: %+v", cfg)
	}
}
