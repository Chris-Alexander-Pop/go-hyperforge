package validator

import (
	"testing"
)

func TestDetectPathTraversal_Robustness(t *testing.T) {
	payloads := []string{
		"../etc/passwd",
		"..%2fetc/passwd",       // Mixed encoded
		"%2e%2e/etc/passwd",     // Mixed encoded
		"%2e%2e%2fetc/passwd",   // Fully encoded
		"..%5cetc/passwd",       // Windows backslash mixed
		"%252e%252e/etc/passwd", // Double encoded dot, single slash
	}

	for _, p := range payloads {
		if !DetectPathTraversal(p) {
			t.Errorf("Failed to detect path traversal in payload: %s", p)
		}
	}
}
