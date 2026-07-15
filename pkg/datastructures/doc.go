/*
Package datastructures provides a collection of efficient data structures.

This package includes implementations for:
  - Linear: Deque, LinkedList, Stack, Queue (including ring and delay queues)
  - Trees: AVL, B+, Fenwick, LSM memtable, Quad, Radix, R-tree, Segment, Trie, BK
  - Probabilistic: BloomFilter (incl. cuckoo/scalable), HyperLogLog, Count-Min Sketch
  - Distributed: CRDT (G-Counter, PN-Counter, G-Set, LWW-Register), VectorClock, Merkle
  - Caching: LRU, LFU, ARC
  - Graph: directed adjacency-list graphs and DAG with topological sort
  - Other: SkipList, Heap, Set/Union-Find, ConcurrentMap, Timer wheel, Suffix array

Preferred reuse (do not reinvent local copies elsewhere in pkg/):

  - heap — priority queues in algorithms (dijkstra, astar, prim, …)
  - bloomfilter — negative-cache / dedup (cache.Bloom, messaging dedup)
  - lru — bounded hot caches (workflow memory definition cache, …)

Maturity varies by subpackage. Several sketch/bitmap/queue variants are marked
experimental in their package docs and should be treated as building blocks,
not production-hardened libraries.
*/
package datastructures
