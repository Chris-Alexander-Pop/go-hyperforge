package otp

import (
	"testing"
	"time"
)

func BenchmarkGenerateCode(b *testing.B) {
	totp := NewTOTP(DefaultTOTPConfig())
	secret, _ := totp.GenerateSecret()
	now := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		totp.GenerateCodeAt(secret, now)
	}
}
