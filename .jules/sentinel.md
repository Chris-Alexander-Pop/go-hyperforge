# Sentinel's Journal

## 2026-01-31 - Path Traversal via URL Encoding
**Vulnerability:** The `SanitizePath` function failed to handle URL-encoded path traversal sequences (e.g., `%2e%2e%2f`), allowing attackers to bypass the sanitization check. It also missed trailing traversal components (e.g., `foo/..`).
**Learning:** `strings.Contains` and `ReplaceAll` on raw input are insufficient for security checks when inputs can be encoded. The `DetectPathTraversal` function knew about these patterns, but the `SanitizePath` function did not use that knowledge.
**Prevention:** Always normalize (decode/clean) inputs before applying security checks or sanitization logic. Use standard libraries (`net/url`, `filepath.Clean`) where possible, or ensure parity between detection and sanitization logic.

## 2026-02-01 - Plaintext Storage via Misleading Variable Names
**Vulnerability:** The `GenerateCodes` function returned raw recovery codes in a variable named `hashedCodes`, leading consumers (like Redis/Memory adapters) to store them in plaintext, assuming they were already hashed.
**Learning:** Variable names like `hashedCodes` can create a false sense of security if the underlying implementation is incomplete or deferred (marked with TODOs).
**Prevention:** Verify that security-critical data flows (like hashing) are actually implemented, not just named as such. Use types (e.g., `HashedCode` vs `PlainCode`) to enforce distinction at compile time if possible.
