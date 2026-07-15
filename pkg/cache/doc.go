/*
Package cache provides a unified caching interface with multiple backend support.

Supported backends:
  - Memory: In-memory cache for testing
  - Redis: Distributed cache (standalone or Cluster via Config.Cluster / Addrs)
  - Bloom: Local bloom filter wrapper

NewFromConfig constructs drivers registered via RegisterDriver (import
adapters/memory and/or adapters/redis). Extended APIs: Exists, MGet, MSet,
Expire, GetTTL. InvalidatePrefix deletes keys by prefix on supporting backends.

Redis Cluster: set Config.Cluster=true and Config.Addrs (seed nodes). DB is
ignored in cluster mode; multi-key ops require same-slot keys.
*/
package cache
