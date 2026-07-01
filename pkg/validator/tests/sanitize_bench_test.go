package validator_test

import (
	"strings"
	"testing"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

func BenchmarkStripHTMLTags_Regex(b *testing.B) {
	s := validator.NewSanitizer(validator.DefaultSanitizerConfig())
	input := "Hello <b>world</b>! This is a <a href=\"#\">test</a> string with some <i>HTML</i>."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Sanitize(input)
	}
}

func BenchmarkStripHTMLTags_Regex_NoTags(b *testing.B) {
	s := validator.NewSanitizer(validator.DefaultSanitizerConfig())
	input := "Hello world! This is a test string without any HTML."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Sanitize(input)
	}
}

func BenchmarkStripHTMLTags_Manual(b *testing.B) {
	input := "Hello <b>world</b>! This is a <a href=\"#\">test</a> string with some <i>HTML</i>."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stripHTMLTagsManual(input)
	}
}

func stripHTMLTagsManual(input string) string {
	if !strings.Contains(input, "<") {
		return input
	}

	var builder strings.Builder
	builder.Grow(len(input))

	i := 0
	for i < len(input) {
		start := strings.IndexByte(input[i:], '<')
		if start == -1 {
			builder.WriteString(input[i:])
			break
		}
		start += i

		end := strings.IndexByte(input[start:], '>')
		if end == -1 {
			builder.WriteString(input[i:])
			break
		}
		end += start

		builder.WriteString(input[i:start])
		i = end + 1
	}

	return builder.String()
}
