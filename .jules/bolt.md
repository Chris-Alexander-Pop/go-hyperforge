## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2025-05-24 - Circular Buffer Implementation Flaw
**Learning:** Implementing circular buffers with bitwise AND masking (`index & (capacity - 1)`) instead of modulo requires strictly enforcing power-of-2 capacity. The existing implementation failed to enforce this precondition, leading to silent data corruption for arbitrary capacities. Additionally, slice-based queues must explicitly zero out popped elements to prevent memory leaks in Go's GC.
**Action:** Always validate preconditions for low-level bitwise optimizations. When reviewing custom data structures, verify both the algorithm's correctness constraints (e.g., power-of-2) and language-specific memory management details (e.g., pointer clearing).

## 2025-02-13 - [Pre-calculate invariant header value to avoid per-request format string allocation]
**Learning:** In Go HTTP middlewares, invariant values (like configuration limits) should be pre-formatted or pre-calculated outside the request handler closure to avoid repeated allocation and formatting work per request. In performance-critical paths (like HTTP middlewares), `strconv` functions (e.g., `strconv.FormatInt`, `strconv.Itoa`) are preferred over `fmt.Sprintf` for integer-to-string conversion to reduce memory allocations and improve execution speed.
**Action:** When writing or reviewing Go HTTP middlewares, look for opportunities to hoist format strings or use `strconv` instead of `fmt.Sprintf` for integer-to-string conversion.
