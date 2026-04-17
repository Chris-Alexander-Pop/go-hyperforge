## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2025-05-24 - Circular Buffer Implementation Flaw
**Learning:** Implementing circular buffers with bitwise AND masking (`index & (capacity - 1)`) instead of modulo requires strictly enforcing power-of-2 capacity. The existing implementation failed to enforce this precondition, leading to silent data corruption for arbitrary capacities. Additionally, slice-based queues must explicitly zero out popped elements to prevent memory leaks in Go's GC.
**Action:** Always validate preconditions for low-level bitwise optimizations. When reviewing custom data structures, verify both the algorithm's correctness constraints (e.g., power-of-2) and language-specific memory management details (e.g., pointer clearing).

## 2025-05-25 - HTML Tag Stripping Regex Optimization
**Learning:** `regexp.ReplaceAllString` is an expensive operation that allocates memory and executes the regex engine even if the input string does not contain a single match. Since `htmlTagRegex` explicitly looks for `<` and `>`, using a fast-path heuristic like `strings.Contains(input, "<")` allows the function to skip regex evaluation entirely for plain text, reducing execution time from ~214ns to ~10ns for safe strings.
**Action:** Always wrap `regexp.ReplaceAllString` with a cheap heuristic check (like `strings.Contains`) if the vast majority of inputs are expected to be clean and unmodified, especially in hot paths like validation and sanitization.

## 2025-05-26 - Redis Key Generation Optimization
**Learning:** In Go, replacing `fmt.Sprintf` with direct string concatenation and `strconv` for hot paths like database or cache key generation significantly reduces CPU overhead and eliminates reflection-based memory allocations.
**Action:** Use string concatenation (`+`) and `strconv` instead of `fmt.Sprintf` for constructing dynamic keys in cache and database adapters (MFA, session, cache). When benchmarking private package methods, use an `export_test.go` file (e.g., `func (p *Provider) TestKey()`) to safely expose them for tests without modifying the production API visibility.

## 2024-04-16 - Optimize TOTP Padding
**Learning:** Formatting strings using `fmt.Sprintf` is surprisingly slow in high-throughput hot paths like OTP generation. The `fmt.Sprintf` implementation uses reflection and heap allocation to construct output formats at runtime.
**Action:** Replace `fmt.Sprintf` integer zero-padding with a combination of `strconv.FormatInt`, string slice bounds indexing on a pre-allocated zeros string slice, and a `strings.Repeat` fallback. This significantly improves speed and drops allocations.

## 2024-05-26 - Pre-formatting invariant values in HTTP middleware closures
**Learning:** In Go HTTP middlewares, `fmt.Sprintf` introduces unnecessary allocation and CPU overhead. Invariant values (like configuration limits) should be pre-formatted outside the request handler closure to avoid per-request allocations. Dynamic values should use `strconv` functions like `strconv.FormatInt` and `strconv.Itoa`.
**Action:** Always inspect middleware for values that can be pre-calculated or pre-formatted outside of the returned `http.HandlerFunc`. Use `strconv` over `fmt.Sprintf` for high-throughput string manipulation.

## 2025-05-26 - String Concatenation Optimization
**Learning:** In Go, manual character-by-character string concatenation with `+=` in a loop should be replaced with `strings.Split` for fixed delimiters to significantly reduce memory allocations and improve performance, as seen in the ~25x speedup for `splitArgon2Hash` (107 allocs/op -> 1 alloc/op).
**Action:** Always prefer built-in functions like `strings.Split` over custom loops with string concatenation for splitting strings.

## 2025-02-17 - Optimize log redaction allocations with MatchString guard
**Learning:** In Go, `regexp.ReplaceAllString` incurs allocation overhead even when there's no match (due to internal setup). When a heuristic guard (e.g., checking for `@` or 13 digits) passes but the actual regex match fails, `ReplaceAllString` will still perform unnecessary allocations. Using `regexp.MatchString` as an additional guard condition before calling `ReplaceAllString` drops these allocations to 0 for non-matching strings, which is critical in hot paths like log handlers (`pkg/logger/handlers.go`).
**Action:** Always consider wrapping `ReplaceAllString` with a `MatchString` guard in performance-critical paths, especially when the heuristic preceding the regex is broad and may produce frequent false positives that fail the regex anyway.

## 2025-03-17 - Pre-calculate static headers in HTTP Middleware
**Learning:** Calling `strings.Join` inside a frequently executed path (like an HTTP middleware handler) on data that doesn't change (like configuration parameters) causes unnecessary per-request memory allocation and CPU overhead.
**Action:** When writing or modifying HTTP middleware (especially those in hot paths like CORS or Rate Limiting), always inspect the handler closure for operations on static configuration data and hoist them (pre-calculate) to the middleware initialization phase outside the returned handler function.
## 2025-02-23 - Replace fmt.Sprintf with string concatenation and strconv in hot paths
**Learning:** Using `fmt.Sprintf` in very high throughput operations like rate limit key generation introduces reflection overhead and unnecessary allocations.
**Action:** Use string concatenation and `strconv` for primitive types in hot path key constructions.
