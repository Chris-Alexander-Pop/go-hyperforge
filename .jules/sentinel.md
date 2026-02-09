# Sentinel's Journal

## 2026-01-31 - Path Traversal via URL Encoding
**Vulnerability:** The `SanitizePath` function failed to handle URL-encoded path traversal sequences (e.g., `%2e%2e%2f`), allowing attackers to bypass the sanitization check. It also missed trailing traversal components (e.g., `foo/..`).
**Learning:** `strings.Contains` and `ReplaceAll` on raw input are insufficient for security checks when inputs can be encoded. The `DetectPathTraversal` function knew about these patterns, but the `SanitizePath` function did not use that knowledge.
**Prevention:** Always normalize (decode/clean) inputs before applying security checks or sanitization logic. Use standard libraries (`net/url`, `filepath.Clean`) where possible, or ensure parity between detection and sanitization logic.

## 2026-02-01 - Plaintext MFA Recovery Codes
**Vulnerability:** MFA recovery codes were stored in plaintext because `GenerateCodes` returned a variable named `hashedCodes` containing raw hex strings, misleading adapters into storing them as-is.
**Learning:** Misleading variable names in security libraries can cause insecure implementations. Comments alone ("hash before storing") are insufficient protection against misuse.
**Prevention:** Security functions should return safe-by-default values (actually hashed) or use strong types (`PlaintextCode` vs `HashedCode`) to prevent confusion.
