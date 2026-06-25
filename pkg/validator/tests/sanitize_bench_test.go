package validator_test

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"testing"
)

func BenchmarkStripHTMLTags(b *testing.B) {
	sanitizer := validator.NewSanitizer(validator.DefaultSanitizerConfig())
	input := "Hello <b>World</b>! This is a <a href=\"#\">test</a> string with <br> HTML tags."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sanitizer.Sanitize(input)
	}
}
