## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2026-02-20 - Rate Limit Key Optimization
**Learning:** In high-throughput paths like rate limiting, `time.Format` and string concatenation are significantly slower than `strconv.AppendInt` with pre-allocated buffers (~10% speedup).
**Action:** Use `strconv.AppendInt` and byte buffers for constructing internal cache keys instead of `fmt.Sprintf` or `time.Format`.
