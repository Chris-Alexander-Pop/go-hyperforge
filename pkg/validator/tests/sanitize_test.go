package validator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
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

func (s *SanitizeSuite) TestSanitizeMap() {
	sanitizer := validator.NewSanitizer(validator.DefaultSanitizerConfig())

	input := map[string]interface{}{
		"name": "<b>Alice</b>",
		"nested": map[string]interface{}{
			"bio": "<script>alert(1)</script>hi",
		},
		"tags": []interface{}{"<i>a</i>", 42},
		"age":  30,
	}

	got := sanitizer.SanitizeMap(input)

	s.Equal("Alice", got["name"])
	nested := got["nested"].(map[string]interface{})
	s.Equal("alert(1)hi", nested["bio"])
	tags := got["tags"].([]interface{})
	s.Equal("a", tags[0])
	s.Equal(42, tags[1])
	s.Equal(30, got["age"])
}

func (s *SanitizeSuite) TestDetectSQLInjection() {
	dropPayload := "DROP" + " TABLE users;"
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"UnionSelect", "1' UNION SELECT * FROM users--", true},
		{"OrEquals", "' OR 1=1 --", true},
		{"DropTable", dropPayload, true},
		{"SelectFrom", "SELECT id FROM accounts", true},
		{"SafeName", "Alice Smith", false},
		{"SafeEmail", "alice@example.com", false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expected, validator.DetectSQLInjection(tt.input), "Input: %s", tt.input)
		})
	}
}

func (s *SanitizeSuite) TestDetectCommandInjection() {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Semicolon", "file.txt; rm -rf /", true},
		{"Pipe", "cat file | grep secret", true},
		{"Backtick", "echo `whoami`", true},
		{"Dollar", "echo $HOME", true},
		{"SafeFilename", "report-2024.pdf", false},
		{"SafeSlug", "hello-world", false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expected, validator.DetectCommandInjection(tt.input), "Input: %s", tt.input)
		})
	}
}

func (s *SanitizeSuite) TestValidatePathInside() {
	base := s.T().TempDir()
	sub := filepath.Join(base, "subdir")
	s.Require().NoError(os.MkdirAll(sub, 0o755))

	ok, err := validator.ValidatePathInside(base, "subdir/file.txt")
	s.NoError(err)
	s.Equal(filepath.Join(base, "subdir", "file.txt"), ok)

	_, err = validator.ValidatePathInside(base, "../outside.txt")
	s.Error(err)
	s.True(errors.IsCode(err, errors.CodeInvalidArgument))
}
