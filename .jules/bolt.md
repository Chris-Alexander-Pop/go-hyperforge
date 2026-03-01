## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2024-05-23 - Fmt.Sprintf vs Strconv/Concatenation in Middleware
**Learning:** `fmt.Sprintf` uses reflection to parse and format strings, adding ~3-4x overhead and massive allocations in high-throughput hot paths (like `RateLimitMiddleware` and `CacheMiddleware`).
**Action:** Always prefer `strconv.FormatInt` or `strconv.Itoa` for integer formatting and simple string concatenation `+` over `fmt.Sprintf` on the hot paths where string building is basic.
