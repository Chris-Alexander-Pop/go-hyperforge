# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to: security@example.com

Include the following information:

- Type of vulnerability (e.g., buffer overflow, SQL injection, XSS)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

## Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Resolution Target**: Within 30 days for critical issues

## Security Best Practices

When using this library:

1. **Keep dependencies updated** - Run `go mod tidy` regularly
2. **Use secrets management** - Never hardcode credentials
3. **Enable TLS** - Use encrypted connections for all adapters
4. **Validate inputs** - Use `pkg/validator` for all user inputs
5. **Handle errors** - Use `pkg/errors` with appropriate error codes

## Security Features

This library includes:

- **`pkg/security/crypto`** - AES-GCM encryption, secure hashing
- **`pkg/security/secrets`** - Secrets management (Vault, AWS, GCP, Azure)
- **`pkg/auth`** - JWT, OAuth2 AS (memory token issuance), OIDC verify/exchange, MFA, WebAuthn
- **`pkg/validator`** - Input validation and sanitization
