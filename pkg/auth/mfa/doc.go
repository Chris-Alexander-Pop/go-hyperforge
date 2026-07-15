// Package mfa provides Multi-Factor Authentication capabilities.
//
// This package supports various MFA methods including:
//   - TOTP (Time-based One-Time Password) via Provider (memory/redis adapters)
//   - SMS/Email OTP via ChannelProvider (adapters/sms, adapters/email)
//   - Recovery codes (SHA-256 hashed at rest; high-entropy one-time codes)
//
// Set Config.EncryptionKey so TOTP secrets are encrypted at rest (AES-GCM via
// pkg/security/crypto). Without it, TOTP secrets remain plaintext in the store.
// Password-like credentials elsewhere in auth use pkg/security/crypto.Hasher
// (see pkg/auth/password and oauth2 memory client secrets).
//
// TOTP usage:
//
//	mfaService, err := memory.New(mfa.Config{TOTPIssuer: "MyApp", EncryptionKey: key})
//	secret, recovery, err := mfaService.Enroll(ctx, userID)
//	ok, err := mfaService.Verify(ctx, userID, code)
//
// SMS (Twilio) / email channel path:
//
//	smsSender, _ := twilio.New(sms.Config{...}) // or sms/adapters/memory for tests
//	channel, _ := mfasms.New(smsSender, mfa.Config{})
//	recovery, _ := channel.Enroll(ctx, userID, "+15551212")
//	_ = channel.CompleteEnrollment(ctx, userID, codeFromSMS)
//	_ = channel.SendChallenge(ctx, userID)
//	ok, _ := channel.Verify(ctx, userID, codeFromSMS)
package mfa
