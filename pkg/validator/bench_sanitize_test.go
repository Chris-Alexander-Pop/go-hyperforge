package validator

import (
	"testing"
)

func BenchmarkStripHTMLTags(b *testing.B) {
	input := "This is a <a href=\"https://example.com\">link</a> and some <b>bold</b> text. <incomplete tag"
	for i := 0; i < b.N; i++ {
		_ = stripHTMLTags(input)
	}
}
