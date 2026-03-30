## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2025-05-24 - Circular Buffer Implementation Flaw
**Learning:** Implementing circular buffers with bitwise AND masking (`index & (capacity - 1)`) instead of modulo requires strictly enforcing power-of-2 capacity. The existing implementation failed to enforce this precondition, leading to silent data corruption for arbitrary capacities. Additionally, slice-based queues must explicitly zero out popped elements to prevent memory leaks in Go's GC.
**Action:** Always validate preconditions for low-level bitwise optimizations. When reviewing custom data structures, verify both the algorithm's correctness constraints (e.g., power-of-2) and language-specific memory management details (e.g., pointer clearing).

## 2025-05-25 - HTML Tag Stripping Regex Optimization
**Learning:** `regexp.ReplaceAllString` is an expensive operation that allocates memory and executes the regex engine even if the input string does not contain a single match. Since `htmlTagRegex` explicitly looks for `<` and `>`, using a fast-path heuristic like `strings.Contains(input, "<")` allows the function to skip regex evaluation entirely for plain text, reducing execution time from ~214ns to ~10ns for safe strings.
**Action:** Always wrap `regexp.ReplaceAllString` with a cheap heuristic check (like `strings.Contains`) if the vast majority of inputs are expected to be clean and unmodified, especially in hot paths like validation and sanitization.
## 2025-03-30 - [Optimize TOTP Zero Padding]
**Learning:** In Go, zero-padding integers in performance-critical paths (like OTP generation) is significantly faster and allocation-free when using `strconv.FormatInt` combined with a pre-allocated zeros string slice rather than `fmt.Sprintf` with dynamic padding formats.
**Action:** Avoid `fmt.Sprintf` for simple string padding and use `strconv` and string manipulation when possible.
