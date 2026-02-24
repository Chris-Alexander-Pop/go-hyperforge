## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).

## 2024-05-24 - sync.Map LoadOrStore Allocation Trap
**Learning:** `sync.Map.LoadOrStore(key, value)` evaluates and allocates `value` *before* the call, even if the key exists. In hot paths like rate limiting, this causes massive unnecessary allocations (structs, mutexes).
**Action:** Use `Load` first. Only if `Load` fails, construct the value and call `LoadOrStore`. This optimizes for the hot path (key exists).
