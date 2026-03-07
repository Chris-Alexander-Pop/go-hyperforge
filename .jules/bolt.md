## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2025-05-24 - Circular Buffer Implementation Flaw
**Learning:** Implementing circular buffers with bitwise AND masking (`index & (capacity - 1)`) instead of modulo requires strictly enforcing power-of-2 capacity. The existing implementation failed to enforce this precondition, leading to silent data corruption for arbitrary capacities. Additionally, slice-based queues must explicitly zero out popped elements to prevent memory leaks in Go's GC.
**Action:** Always validate preconditions for low-level bitwise optimizations. When reviewing custom data structures, verify both the algorithm's correctness constraints (e.g., power-of-2) and language-specific memory management details (e.g., pointer clearing).

## 2025-03-07 - [Replaced fmt.Sprintf with strconv in RateLimitMiddleware]
**Learning:** In highly trafficked HTTP middlewares like RateLimitMiddleware, using `fmt.Sprintf` for formatting numeric headers (e.g. `X-RateLimit-Limit`) causes unnecessary memory allocations and CPU overhead due to reflection. `fmt.Sprintf("%d", ...)` runs on every request for 3 different headers.
**Action:** Replaced `fmt.Sprintf` with `strconv.FormatInt(..., 10)` and `strconv.Itoa(...)` which reduced allocations by 20% (10 -> 8 per request) and improved execution speed by ~33% (from 1254 ns/op down to 835.7 ns/op), proving that `strconv` is essential for hot-path numeric formatting.
