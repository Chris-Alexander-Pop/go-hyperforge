# Sentinel's Journal

## 2026-01-31 - Path Traversal via URL Encoding
**Vulnerability:** The `SanitizePath` function failed to handle URL-encoded path traversal sequences (e.g., `%2e%2e%2f`), allowing attackers to bypass the sanitization check. It also missed trailing traversal components (e.g., `foo/..`).
**Learning:** `strings.Contains` and `ReplaceAll` on raw input are insufficient for security checks when inputs can be encoded. The `DetectPathTraversal` function knew about these patterns, but the `SanitizePath` function did not use that knowledge.
**Prevention:** Always normalize (decode/clean) inputs before applying security checks or sanitization logic. Use standard libraries (`net/url`, `filepath.Clean`) where possible, or ensure parity between detection and sanitization logic.

## 2026-02-01 - Plaintext Storage via Misleading Variable Names
**Vulnerability:** The `GenerateCodes` function returned raw recovery codes in a variable named `hashedCodes`, leading consumers (like Redis/Memory adapters) to store them in plaintext, assuming they were already hashed.
**Learning:** Variable names like `hashedCodes` can create a false sense of security if the underlying implementation is incomplete or deferred (marked with TODOs).
**Prevention:** Verify that security-critical data flows (like hashing) are actually implemented, not just named as such. Use types (e.g., `HashedCode` vs `PlainCode`) to enforce distinction at compile time if possible.

## 2026-10-18 - Path Traversal in Local Storage
**Vulnerability:** The local blob storage adapter used `filepath.Join` without validating that the resulting path remained within the base directory. This allowed attackers to access arbitrary files on the system using `../` sequences.
**Learning:** `filepath.Join` cleans paths but does not enforce a sandbox. Relying on relative paths for the sandbox root (e.g., `.`) can fail if not resolved to an absolute path before prefix checking, as `filepath.Join` might optimize `.` away.
**Prevention:** Resolve the sandbox root to an absolute path using `filepath.Abs`. Verify the resolved path starts with the sandbox root (ensure trailing separator is handled) before performing any file operations.
