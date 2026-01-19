/*
Package distlock provides distributed locking interfaces and implementations.

Supported backends:
  - Memory: Local in-memory lock (for testing/single-node)
  - Redis: Redis-based distributed lock (Redlock or simple SET NX)
*/
package distlock
