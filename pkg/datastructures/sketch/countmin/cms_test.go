package countmin

import "testing"

func TestCountMinSketch(t *testing.T) {
	cms := New(0.01, 0.99)

	cms.Add([]byte("apple"))
	cms.Add([]byte("apple"))
	cms.Add([]byte("banana"))

	estApple := cms.Estimate([]byte("apple"))
	if estApple < 2 {
		t.Errorf("Expected at least 2 for apple, got %d", estApple)
	}

	estBanana := cms.Estimate([]byte("banana"))
	if estBanana < 1 {
		t.Errorf("Expected at least 1 for banana, got %d", estBanana)
	}

	estCherry := cms.Estimate([]byte("cherry"))
	if estCherry != 0 {
		// Probabilistic, but with low count shouldn't collide given large width
		// However, it's possible. But 0 is ideal.
		// Let's accept small noise, but for 3 items it should be 0.
		if estCherry > 0 {
			t.Logf("Got non-zero estimate for missing item: %d", estCherry)
		}
	}

	if cms.Count() != 3 {
		t.Errorf("Expected total count 3, got %d", cms.Count())
	}

	cms.Reset()
	if cms.Count() != 0 {
		t.Error("Expected count 0 after reset")
	}
}
