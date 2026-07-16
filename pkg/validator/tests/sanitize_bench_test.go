package validator_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
)

func BenchmarkSanitizeStripHTML(b *testing.B) {
	config := validator.DefaultSanitizerConfig()
	sanitizer := validator.NewSanitizer(config)
	input := "This is <b>bold</b> and <i>italic</i> with <script>alert('xss')</script> and some <unclosed tag"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sanitizer.Sanitize(input)
	}
}
