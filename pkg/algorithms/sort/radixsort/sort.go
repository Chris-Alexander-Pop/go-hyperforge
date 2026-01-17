package radixsort

// Sort sorts a slice of integers using Radix Sort (LSD).
// Time Complexity: O(nk) where k is number of digits. Stable.
// Note: This implementation assumes non-negative integers for simplicity.
func Sort(s []int) {
	if len(s) == 0 {
		return
	}

	maxVal := getMax(s)

	// Do counting sort for every digit. Exponent is 10^i
	for exp := 1; maxVal/exp > 0; exp *= 10 {
		countingSort(s, exp)
	}
}

func getMax(s []int) int {
	mx := s[0]
	for _, v := range s {
		if v > mx {
			mx = v
		}
	}
	return mx
}

func countingSort(s []int, exp int) {
	n := len(s)
	output := make([]int, n)
	count := make([]int, 10)

	// Store count of occurrences in count[]
	for i := 0; i < n; i++ {
		count[(s[i]/exp)%10]++
	}

	// Change count[i] so that count[i] now contains actual
	// position of this digit in output[]
	for i := 1; i < 10; i++ {
		count[i] += count[i-1]
	}

	// Build the output array
	for i := n - 1; i >= 0; i-- {
		output[count[(s[i]/exp)%10]-1] = s[i]
		count[(s[i]/exp)%10]--
	}

	// Copy the output array to s[], so that s[] now
	// contains numbers sorted according to current digit
	for i := 0; i < n; i++ {
		s[i] = output[i]
	}
}
