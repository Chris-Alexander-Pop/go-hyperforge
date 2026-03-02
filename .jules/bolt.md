## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2024-03-02 - Middleware Format Integer Optimization
**Learning:** `fmt.Sprintf("%d", ...)` creates unnecessary memory allocations for integer-to-string formatting, particularly detrimental in high-throughput HTTP middleware hot paths.
**Action:** Use `strconv.FormatInt` and `strconv.Itoa` to eliminate formatting allocations and improve HTTP request processing speed in performance-critical paths like rate limiting headers.
