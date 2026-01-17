# Concurrency & Resilience

## Critical Reliability Features
- [x] **Rate Limiter**:
    - **Token Bucket**: Burstable traffic control.
    - **Leaky Bucket**: Smooth traffic flow.
    - **Sliding Window**: Precise throttling.
    - **Fixed Window**: Simple time-bucketed counter.
    - Distributed Rate Limiting via Redis Lua scripts.
- [x] **Circuit Breaker**:
    - State Machine (Open/Closed/Half-Open).
    - Failure threshold detection.
    - Automatic recovery with half-open probing.
- [x] **Consistent Hashing**:
    - Hash Ring implementation for distributed load balancing.
    - Virtual Nodes for skew minimization.

## Concurrency Patterns
- [x] **Worker Pools**: Fixed-size goroutine pools.
- [x] **Pipelines**: Stream processing stages (Fan-out / Fan-in / Batch / Filter / Map).
- [x] **Semaphores**: Limiting concurrent access to resources.
- [x] **Distributed Locks**: Redis-based implementation with atomic operations.
- [x] **Smart Mutexes**: Observability-enhanced mutexes with deadlock detection.

## Optimization
- [x] **Bloom Filters**: Probabilistic set checking (Database avoidance).
- [x] **HyperLogLog**: Cardinality estimation.
- [x] **Sharded Map**: Lock-free concurrent map with sharding.
