## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2025-05-24 - Circular Buffer Implementation Flaw
**Learning:** Implementing circular buffers with bitwise AND masking (`index & (capacity - 1)`) instead of modulo requires strictly enforcing power-of-2 capacity. The existing implementation failed to enforce this precondition, leading to silent data corruption for arbitrary capacities. Additionally, slice-based queues must explicitly zero out popped elements to prevent memory leaks in Go's GC.
**Action:** Always validate preconditions for low-level bitwise optimizations. When reviewing custom data structures, verify both the algorithm's correctness constraints (e.g., power-of-2) and language-specific memory management details (e.g., pointer clearing).

## 2025-05-25 - Avoid fmt.Sprintf in High-Throughput HTTP Middleware
**Learning:** `fmt.Sprintf` uses reflection under the hood and performs numerous small allocations, severely degrading latency in hot paths like HTTP middleware. By simply replacing `fmt.Sprintf("%d", value)` with `strconv.FormatInt(value, 10)` or `strconv.Itoa(value)`, we eliminated 2 allocations per rate-limited request, reducing latency by ~30% per loop. String concatenation (`"prefix:" + val`) is similarly superior to `fmt.Sprintf`.
**Action:** Aggressively hunt down and eliminate `fmt.Sprintf` in core APIs, middleware, and request handling loops, opting for `strconv` and string concatenation instead.
