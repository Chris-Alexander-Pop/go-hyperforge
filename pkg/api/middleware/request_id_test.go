package middleware

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateRequestID_Randomness(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		id := generateRequestID()

		// Check for duplicates
		if ids[id] {
			t.Fatalf("Collision detected! ID %s was generated twice", id)
		}
		ids[id] = true

		// Check format (UUID)
		_, err := uuid.Parse(id)
		assert.NoError(t, err, "Generated ID should be a valid UUID")
	}

	assert.Equal(t, count, len(ids), "Should have unique IDs")
}
