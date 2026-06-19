package validator

import (
	"regexp"
	"strings"
	"testing"
)

// Legacy implementation for benchmarking comparison
var legacyHTMLTagRegex = regexp.MustCompile(`<[^>]*>`)

func legacyStripHTMLTags(input string) string {
	if !strings.Contains(input, "<") {
		return input
	}
	return legacyHTMLTagRegex.ReplaceAllString(input, "")
}

func BenchmarkStripHTMLRegex(b *testing.B) {
	input := "Hello <b>World</b>! This is a <a href=\"#\">test</a> string with <unclosed tag"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		legacyStripHTMLTags(input)
	}
}

func BenchmarkStripHTMLManual(b *testing.B) {
	input := "Hello <b>World</b>! This is a <a href=\"#\">test</a> string with <unclosed tag"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stripHTMLTags(input)
	}
}

func TestStripHTMLMatch(t *testing.T) {
	inputs := []string{
		"Hello <b>World</b>!",
		"No tags here",
		"Start <tag> and </tag> end",
		"<p>multiple</p> <div>tags</div>",
		"Unclosed <tag without end",
		"<unclosed",
		"Multiple <unclosed <tags",
		"<> empty tag",
	}

	for _, input := range inputs {
		regexRes := legacyStripHTMLTags(input)
		manualRes := stripHTMLTags(input)
		if regexRes != manualRes {
			t.Errorf("Mismatch for %q:\nRegex : %q\nManual: %q", input, regexRes, manualRes)
		}
	}
}
