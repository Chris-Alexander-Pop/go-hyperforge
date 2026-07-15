// Package mfa provides Multi-Factor Authentication capabilities.
//
// This package supports various MFA methods including:
//   - TOTP (Time-based One-Time Password)
//   - SMS/Email OTP (via communication package)
//   - Recovery codes
//
// Usage:
//
//	mfaService, err := memory.New(mfa.Config{TOTPIssuer: "MyApp"})
//	secret, recovery, err := mfaService.Enroll(ctx, userID)
//	ok, err := mfaService.Verify(ctx, userID, code)
package mfa
