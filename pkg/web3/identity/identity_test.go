package identity

import "testing"

func BenchmarkParseDID(b *testing.B) {
	did := "did:ethr:0x1234567890123456789012345678901234567890"
	for i := 0; i < b.N; i++ {
		_, _ = ParseDID(did)
	}
}
