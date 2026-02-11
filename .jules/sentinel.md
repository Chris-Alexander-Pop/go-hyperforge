# Sentinel's Journal

## 2026-01-31 - Path Traversal via URL Encoding
**Vulnerability:** The `SanitizePath` function failed to handle URL-encoded path traversal sequences (e.g., `%2e%2e%2f`), allowing attackers to bypass the sanitization check. It also missed trailing traversal components (e.g., `foo/..`).
**Learning:** `strings.Contains` and `ReplaceAll` on raw input are insufficient for security checks when inputs can be encoded. The `DetectPathTraversal` function knew about these patterns, but the `SanitizePath` function did not use that knowledge.
**Prevention:** Always normalize (decode/clean) inputs before applying security checks or sanitization logic. Use standard libraries (`net/url`, `filepath.Clean`) where possible, or ensure parity between detection and sanitization logic.

## 2026-05-24 - Malformed Security Headers via Incorrect Type Conversions
**Vulnerability:** The `SecurityHeaders` and `CORS` middleware produced invalid headers because `string(rune(int))` was used to convert integers to strings (resulting in garbage characters) and `time.Duration.String()` was used for seconds (resulting in nanosecond units like "86.4µs").
**Learning:** In Go, `string(myInt)` does not convert the number to a string; it converts it to a Unicode code point. Similarly, `time.Duration(seconds)` interprets the value as nanoseconds.
**Prevention:** Always use `strconv.Itoa(i)` or `fmt.Sprintf("%d", i)` for integer-to-string conversion. For time durations representing seconds, ensure the value is correctly scaled or formatted as an integer if the header expects seconds.
