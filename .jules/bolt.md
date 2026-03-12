## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2025-05-24 - Circular Buffer Implementation Flaw
**Learning:** Implementing circular buffers with bitwise AND masking (`index & (capacity - 1)`) instead of modulo requires strictly enforcing power-of-2 capacity. The existing implementation failed to enforce this precondition, leading to silent data corruption for arbitrary capacities. Additionally, slice-based queues must explicitly zero out popped elements to prevent memory leaks in Go's GC.
**Action:** Always validate preconditions for low-level bitwise optimizations. When reviewing custom data structures, verify both the algorithm's correctness constraints (e.g., power-of-2) and language-specific memory management details (e.g., pointer clearing).

## 2025-05-25 - HTTP Middleware `fmt.Sprintf` Allocation Overhead
**Learning:** Using `fmt.Sprintf` for simple integer-to-string conversions (e.g., `fmt.Sprintf("%d", limit)`) inside high-throughput hot paths like HTTP middleware generates significant, unnecessary memory allocations (9 allocs/op vs 6 allocs/op in benchmarks).
**Action:** Use `strconv.FormatInt` or `strconv.Itoa` for simple number conversions in performance-critical code. Additionally, invariant values (like configuration limits) should be pre-formatted outside the request handler closure to avoid repeated work per request.
