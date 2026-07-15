package segment_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/tree/segment"
	"testing"
)

func TestSegmentTree(t *testing.T) {
	nums := []int{1, 3, 5, 7, 9, 11}
	st := segment.New(nums, func(a, b int) int { return a + b }) // Sum Query

	if sum := st.Query(0, 2); sum != 9 { // 1 + 3 + 5
		t.Errorf("Expected sum 9, got %d", sum)
	}

	st.Update(1, 10)                      // nums[1] = 10 -> {1, 10, 5, 7, 9, 11}
	if sum := st.Query(0, 2); sum != 16 { // 1 + 10 + 5
		t.Errorf("Expected sum 16, got %d", sum)
	}
}
