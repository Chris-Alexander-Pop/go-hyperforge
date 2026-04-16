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

## 2026-02-05 - Path Traversal in URL Generation
**Vulnerability:** The `URL` method in the local blob storage adapter constructed URLs by blindly concatenating the base directory with the user-provided key using `filepath.Join`. This allowed attackers to generate valid `file://` URLs pointing to arbitrary files on the system (e.g., `file:///etc/passwd`) by using path traversal sequences like `../`.
**Learning:** Security validation must be applied consistently across ALL methods that handle user input, not just data access methods like `Upload` or `Download`. Auxiliary methods that generate references (URLs, paths) are equally critical if their output is trusted by consumers.
**Prevention:** Ensure that URL generation logic reuses the same path validation and sanitization routines as the core storage operations. Treat the generation of a file path reference as a security-sensitive operation.
## 2024-05-24 - Fail Securely in Cryptography & Time Constant Compare
**Vulnerability:** The CSRF token generation in `pkg/api/middleware/security.go` fell back to a highly predictable timestamp string (`time.Now().String()`) if the system's cryptographically secure pseudo-random number generator (`crypto/rand`) failed. Furthermore, token validation used standard string inequality (`!=`), making it vulnerable to timing attacks.
**Learning:** In cryptographic or security contexts, if a dependency like a random number generator fails, the application must "fail securely" (e.g., panic or return an error that halts the operation), rather than silently falling back to insecure, predictable defaults. Additionally, sensitive token comparisons must use constant-time operations like `crypto/subtle.ConstantTimeCompare` to prevent information leakage.
**Prevention:** Audit all uses of `crypto/rand` to ensure errors are not masked by predictable fallbacks. Enforce the use of `crypto/subtle.ConstantTimeCompare` for all sensitive token or hash comparisons across the codebase.

## 2024-04-14 - Missing Command Injection Validation in Global Middleware
**Vulnerability:** The `SanitizeMiddleware` checked for SQL injection and Path Traversal but completely omitted checking for Command Injection, leaving a massive gap considering parts of the codebase execute underlying processes via `exec.CommandContext`.
**Learning:** Security validations must be comprehensive, especially global middlewares, and missing checks can lead to critical vulnerabilities being propagated throughout the entire application.
**Prevention:** Regularly audit the capabilities of the validation and sanitization packages against the usage patterns in global middlewares.

## 2024-05-24 - Timing Attack in MFA Recovery Code Comparison
**Vulnerability:** The MFA recovery code validation in `pkg/auth/mfa/adapters/memory/memory.go` and `pkg/auth/mfa/adapters/redis/redis.go` used standard string inequality (`hash == hashedCode`), making it vulnerable to timing attacks.
**Learning:** In cryptographic or security contexts, sensitive token comparisons must use constant-time operations like `crypto/subtle.ConstantTimeCompare` to prevent information leakage.
**Prevention:** Enforce the use of `crypto/subtle.ConstantTimeCompare` for all sensitive token or hash comparisons across the codebase.

## 2026-02-05 - Host Header Injection in Middlewares
**Vulnerability:** The `RequireHTTPS` middleware constructed absolute redirect URLs blindly trusting the `r.Host` value. This allowed an attacker to supply an arbitrary `Host` header (e.g. `evil.com`), tricking the server into issuing a 301 redirect to an attacker-controlled site, potentially leading to phishing or token leakage.
**Learning:** `r.Host` is user-supplied data and must never be trusted implicitly when constructing absolute URLs, especially in security boundaries like HTTP-to-HTTPS redirects.
**Prevention:** Introduce validation for the `Host` header against an explicit whitelist of allowed hosts. For backward compatibility where a whitelist isn't provided, enforce strict character validation (e.g., alphanumeric, dots, dashes) to prevent structural attacks like path traversal or query string injection via the Host header.
