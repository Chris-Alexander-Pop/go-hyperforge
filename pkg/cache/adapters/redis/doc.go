// Package redis provides a Redis cache adapter.
//
// Importing this package registers the "redis" driver for cache.NewFromConfig.
// Prefer NewWithClient for miniredis unit tests.
//
// Cluster mode: set cache.Config.Cluster=true with Addrs (or Host:Port seed),
// or call NewCluster. DB is ignored in cluster mode; MGet/MSet need same-slot keys.
package redis
