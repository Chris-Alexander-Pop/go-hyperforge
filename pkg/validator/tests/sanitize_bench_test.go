package validator_test

import (
	"regexp"
	"strings"
	"testing"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func regexStrip(input string) string {
	if !strings.Contains(input, "<") {
		return input
	}
	return htmlTagRegex.ReplaceAllString(input, "")
}

func manualStrip(input string) string {
	if !strings.Contains(input, "<") {
		return input
	}
	var sb strings.Builder
	sb.Grow(len(input))
	i := 0
	for i < len(input) {
		idx := strings.IndexByte(input[i:], '<')
		if idx == -1 {
			sb.WriteString(input[i:])
			break
		}
		sb.WriteString(input[i : i+idx])
		i += idx
		endIdx := strings.IndexByte(input[i:], '>')
		if endIdx == -1 {
			sb.WriteString(input[i:])
			break
		}
		i += endIdx + 1
	}
	return sb.String()
}

func BenchmarkRegex(b *testing.B) {
	input := "This is a <test> string with <b>some</b> HTML <br> tags in it to see <a href='foo'>performance</a>."
	for i := 0; i < b.N; i++ {
		regexStrip(input)
	}
}

func BenchmarkManual(b *testing.B) {
	input := "This is a <test> string with <b>some</b> HTML <br> tags in it to see <a href='foo'>performance</a>."
	for i := 0; i < b.N; i++ {
		manualStrip(input)
	}
}
