package identity

import (
	"testing"
)

func TestParseDID(t *testing.T) {
	d, err := ParseDID("did:ethr:0x123")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if d.Method != "ethr" || d.Identifier != "0x123" {
		t.Errorf("unexpected parsed DID: %+v", d)
	}
}

func BenchmarkParseDID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseDID("did:ethr:0x1234567890abcdef1234567890abcdef12345678/path/to/resource?query=1#fragment")
	}
}
