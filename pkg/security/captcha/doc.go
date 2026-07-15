// Package captcha provides bot protection interfaces.
//
// Implementations:
//   - adapters/memory — deterministic token check for tests
//   - adapters/recaptcha — Google reCAPTCHA v2/v3 siteverify HTTP client
//
// hCaptcha / Turnstile are reserved Provider names without adapters yet.
package captcha
