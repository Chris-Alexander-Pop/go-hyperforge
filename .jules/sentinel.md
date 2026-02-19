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

## 2026-02-01 - Malformed Security Headers via Incorrect Type Conversion
**Vulnerability:** The `HSTS` and `CORS` middlewares generated invalid headers (e.g., `Strict-Transport-Security: max-age=` and `Access-Control-Max-Age: 86.4µs`) due to incorrect integer-to-string conversions. `string(rune(int))` creates a unicode character from the integer value, not a string representation of the number. `time.Duration(int).String()` formats as a duration string with units, not raw seconds.
**Learning:** Type conversion in Go requires care. `string(int)` treats the integer as a rune, and `time.Duration`'s string representation is human-readable, not protocol-compliant for raw seconds. The lack of unit tests for these middleware components allowed the bug to remain undetected.
**Prevention:** Use `strconv.Itoa` or `fmt.Sprintf` for number-to-string conversion. Always implement unit tests that assert the exact string value of security headers to ensure compliance with HTTP specifications.

## 2026-02-04 - SQL Injection in DDL Construction
**Vulnerability:** The `CreateRangePartition` function in `pkg/database/partitioning/ddl.go` constructed SQL queries using `fmt.Sprintf` with user-supplied strings inside single quotes (`'%s'`). This allowed attackers to escape the string literal via a single quote and inject arbitrary SQL.
**Learning:** DDL statements (like `CREATE TABLE`) often don't support parameterized queries (prepared statements) for all values, leading developers to fallback to string formatting. This is a common trap.
**Prevention:** When parameterized queries are not possible, ALWAYS use a dedicated escaping function (like `quoteLiteral`) that handles the specific escaping rules of the database dialect (e.g., doubling single quotes). Never trust string concatenation for SQL.

## 2026-02-05 - Rate Limit Bypass via Port Rotation
**Vulnerability:** The `RateLimitMiddleware` used `r.RemoteAddr` (e.g., `192.0.2.1:12345`) directly as the rate limiting key. Since every new TCP connection uses a different source port, attackers could bypass rate limits simply by opening new connections for each request.
**Learning:** In Go, `http.Request.RemoteAddr` contains both the IP address and the port. Security controls that rely on IP identity must explicitly strip the port component using `net.SplitHostPort`.
**Prevention:** Always sanitize and normalize identifiers used for security decisions. For IP-based controls, ensure only the IP address is used. Add regression tests that verify the stability of the identifier across different request variations (e.g., changing ports).
