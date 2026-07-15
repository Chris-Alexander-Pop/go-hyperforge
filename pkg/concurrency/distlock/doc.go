/*
Package distlock provides distributed locking interfaces and implementations.

Supported backends:
  - Memory: Local in-memory lock (for testing/single-node)
  - Redis: Single-instance Redis lock via SET NX + Lua release/extend

This is not Redlock. Multi-node Redis Redlock (majority quorum across
independent Redis masters) is planned and not implemented. Prefer a single
strongly consistent store, or document your failure model carefully if you
need multi-master lock semantics.
*/
package distlock
