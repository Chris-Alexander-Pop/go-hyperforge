package validator

import (
	"testing"
)

func BenchmarkStripHTMLTags(b *testing.B) {
	input := "This is a <script>alert('XSS')</script> test with <b>HTML</b> tags."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stripHTMLTags(input)
	}
}

func BenchmarkStripHTMLTags_NoTags(b *testing.B) {
	input := "This is a clean string without any HTML tags at all."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stripHTMLTags(input)
	}
}
