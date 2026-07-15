/*
Package cache provides a unified caching interface with multiple backend support.

Supported backends:
  - Memory: In-memory cache for testing
  - Redis: Distributed cache
  - Bloom: Local bloom filter wrapper

NewFromConfig constructs drivers registered via RegisterDriver (import
adapters/memory and/or adapters/redis). Extended APIs: Exists, MGet, MSet,
Expire, GetTTL. InvalidatePrefix deletes keys by prefix on supporting backends.
*/
package cache
