/*
Package database provides a unified interface for interacting with various database systems.

Features:
  - Unified Interface: Common abstraction for SQL, NoSQL, and Vector databases.
  - Adapters: Pluggable backends (PostgreSQL, MySQL, Redis, MongoDB, Pinecone, etc.).
  - Capabilities: Sharding (consistent hash + sql.Sharded), Partitioning, Vector Search, Introspection.
  - Resilience: Optional — use ops.WithRetry (pkg/resilience) and sql.NewResilientSQL
    for retries with optional circuit breaking. Not enabled by default on adapters.
*/
package database
