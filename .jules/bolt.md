## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2025-05-24 - Circular Buffer Implementation Flaw
**Learning:** Implementing circular buffers with bitwise AND masking (`index & (capacity - 1)`) instead of modulo requires strictly enforcing power-of-2 capacity. The existing implementation failed to enforce this precondition, leading to silent data corruption for arbitrary capacities. Additionally, slice-based queues must explicitly zero out popped elements to prevent memory leaks in Go's GC.
**Action:** Always validate preconditions for low-level bitwise optimizations. When reviewing custom data structures, verify both the algorithm's correctness constraints (e.g., power-of-2) and language-specific memory management details (e.g., pointer clearing).

## 2025-02-17 - Optimize log redaction allocations with MatchString guard
**Learning:** In Go, `regexp.ReplaceAllString` incurs allocation overhead even when there's no match (due to internal setup). When a heuristic guard (e.g., checking for `@` or 13 digits) passes but the actual regex match fails, `ReplaceAllString` will still perform unnecessary allocations. Using `regexp.MatchString` as an additional guard condition before calling `ReplaceAllString` drops these allocations to 0 for non-matching strings, which is critical in hot paths like log handlers (`pkg/logger/handlers.go`).
**Action:** Always consider wrapping `ReplaceAllString` with a `MatchString` guard in performance-critical paths, especially when the heuristic preceding the regex is broad and may produce frequent false positives that fail the regex anyway.
