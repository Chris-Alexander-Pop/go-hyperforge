## 2024-05-23 - Regex Redaction Optimization
**Learning:** `regexp.ReplaceAllString` executes expensive logic even if no replacement is needed. In high-throughput paths like logging middleware, unconditional execution of regex and unconditional slice allocation for "modified" records creates massive overhead (19 allocs/op -> 0 allocs/op).
**Action:** Use fast-path heuristics (`strings.Contains`, length checks) to skip regex execution. Use a "check-only" pass to detect changes before allocating new structures to avoid allocation in the happy path (no redaction).
## 2025-01-20 - [Load-before-LoadOrStore on sync.Map]
**Learning:** sync.Map.LoadOrStore always allocates the value argument (passed by interface) even if the key exists. Using a Load check first (optimistic read) avoids this expensive allocation on hot paths, especially when constructing complex structs.
**Action:** Always wrap LoadOrStore with a Load check when the value creation is non-trivial or allocates.
