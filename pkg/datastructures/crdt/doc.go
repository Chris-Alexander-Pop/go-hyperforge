/*
Package crdt provides Conflict-free Replicated Data Types.

Supported types:
  - G-Counter (grow-only counter)
  - PN-Counter (positive-negative counter)
  - G-Set (grow-only set)
  - LWW-Register (last-writer-wins register)

These are educational / library building blocks. Prefer battle-tested
CRDT libraries for production multi-region state when stronger guarantees
or richer types (OR-Set, RGA, etc.) are required.
*/
package crdt
