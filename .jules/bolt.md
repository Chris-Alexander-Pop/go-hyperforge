## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2026-03-08 - Bit Manipulation Optimization
**Learning:** `math/bits` package provides hardware-accelerated bit manipulation functions like `LeadingZeros64` which are significantly faster than manual loops. In hot paths like HyperLogLog `Add`, replacing a manual `clz` loop with `math/bits.LeadingZeros64` and inlining FNV-1a hashing reduced execution time by ~43% (25.97ns -> 14.77ns).
**Action:** Prefer `math/bits` for bitwise operations. Inline simple hash functions like FNV-1a in high-performance data structures to avoid interface allocation overhead.
