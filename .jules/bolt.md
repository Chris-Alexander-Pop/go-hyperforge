## 2025-02-28 - [Load before LoadOrStore]
**Learning:** Calling `sync.Map.LoadOrStore` unconditionally allocates heap objects for the value to store (even if the key exists). In performance-critical paths (like rate limiting `Allow` methods), this causes unnecessary allocations per request.
**Action:** Use a "Load-before-LoadOrStore" pattern (`Load` first, and if not found, then `LoadOrStore`) to avoid allocations when the key already exists. Also, use string concatenation (`+`) instead of `fmt.Sprintf` for key generation to reduce allocations.
