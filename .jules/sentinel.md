# Sentinel's Journal

## 2026-01-31 - Path Traversal via URL Encoding
**Vulnerability:** The `SanitizePath` function failed to handle URL-encoded path traversal sequences (e.g., `%2e%2e%2f`), allowing attackers to bypass the sanitization check. It also missed trailing traversal components (e.g., `foo/..`).
**Learning:** `strings.Contains` and `ReplaceAll` on raw input are insufficient for security checks when inputs can be encoded. The `DetectPathTraversal` function knew about these patterns, but the `SanitizePath` function did not use that knowledge.
**Prevention:** Always normalize (decode/clean) inputs before applying security checks or sanitization logic. Use standard libraries (`net/url`, `filepath.Clean`) where possible, or ensure parity between detection and sanitization logic.

## 2026-05-21 - Malformed Security Headers via Invalid Type Conversion
**Vulnerability:** `Strict-Transport-Security` and `Access-Control-Max-Age` headers were malformed because of incorrect type conversions. `string(rune(int))` generated invalid characters instead of stringified numbers, and `time.Duration(int).String()` generated human-readable duration strings (e.g., "86.4µs") instead of raw seconds.
**Learning:** `string(rune(myInt))` is almost always a bug when trying to convert a number to a string; it treats the integer as a Unicode code point. Similarly, `time.Duration(myInt)` interprets the integer as nanoseconds, which is rarely intended for headers expecting seconds.
**Prevention:** Always use `strconv.Itoa()` for integer-to-string conversions. Be explicit about units when working with `time.Duration`.
