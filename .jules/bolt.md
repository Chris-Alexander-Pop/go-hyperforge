## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2024-05-24 - Zero-Allocation Bloom Filter
**Learning:** Returning a slice from a helper function (even if small/fixed size) forces allocation. Inlining the logic allows the compiler to use stack allocation or registers. Also, `[]byte(string)` conversion is optimized to zero-allocation by the Go compiler when the result is used only as a read-only argument to a function that doesn't escape the slice.
**Action:** Inline tight loops that would otherwise return small slices. Use `var buf [N]byte; h.Sum(buf[:0])` instead of `h.Sum(nil)` to guarantee zero allocation for hash sums.
