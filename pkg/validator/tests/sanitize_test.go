package validator_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/test"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

type SanitizeSuite struct {
	*test.Suite
}

func TestSanitizeSuite(t *testing.T) {
	test.Run(t, &SanitizeSuite{Suite: test.NewSuite()})
}

func (s *SanitizeSuite) TestSanitizePath() {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard traversal",
			input:    "../../etc/passwd",
			expected: "etc/passwd",
		},
		{
			name:     "Nested traversal",
			input:    "foo/../bar",
			expected: "foo/bar",
		},
		{
			name:     "Win traversal",
			input:    "..\\..\\windows",
			expected: "windows",
		},
		{
			name:     "Encoded traversal",
			input:    "%2e%2e%2fetc%2fpasswd",
			expected: "etc/passwd",
		},
		{
			name:     "Double encoded traversal",
			input:    "%252e%252e%252fetc%252fpasswd",
			expected: "etc/passwd",
		},
		{
			name:     "Mixed encoded",
			input:    "..%2fetc%2fpasswd",
			expected: "etc/passwd",
		},
		{
			name:     "Trailing dots",
			input:    "foo/..",
			expected: "foo",
		},
		{
			name:     "Trailing dots win",
			input:    "foo\\..",
			expected: "foo",
		},
		{
			name:     "Exact match dots",
			input:    "..",
			expected: "",
		},
		{
			name:     "Valid filename with dots",
			input:    "foo..bar",
			expected: "foo..bar",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := validator.SanitizePath(tt.input)
			s.Equal(tt.expected, got, "Input: %s", tt.input)
		})
	}
}

func (s *SanitizeSuite) TestDetectPathTraversal() {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Standard traversal",
			input:    "../../etc/passwd",
			expected: true,
		},
		{
			name:     "Safe path",
			input:    "etc/passwd",
			expected: false,
		},
		{
			name:     "Encoded traversal",
			input:    "%2e%2e%2fetc%2fpasswd",
			expected: true,
		},
		{
			name:     "Double encoded traversal",
			input:    "%252e%252e%252fetc%252fpasswd",
			expected: true,
		},
		{
			name:     "Triple encoded traversal",
			input:    "%25252e%25252e%25252f",
			expected: true,
		},
		{
			name:     "Mixed encoding",
			input:    "%2e%2e%252f",
			expected: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := validator.DetectPathTraversal(tt.input)
			s.Equal(tt.expected, got, "Input: %s", tt.input)
		})
	}
}
