package validator

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

// Sanitizer sanitizes user input to prevent XSS and injection attacks.
type Sanitizer struct {
	config SanitizerConfig
}

// SanitizerConfig configures the sanitizer.
type SanitizerConfig struct {
	// StripHTML removes all HTML tags.
	StripHTML bool

	// EscapeHTML escapes HTML entities.
	EscapeHTML bool

	// MaxLength limits the maximum length of input.
	MaxLength int

	// AllowedTags is a list of HTML tags to allow (if StripHTML is false).
	AllowedTags []string

	// AllowedAttributes is a map of tag -> allowed attributes.
	AllowedAttributes map[string][]string
}

// DefaultSanitizerConfig returns secure defaults.
func DefaultSanitizerConfig() SanitizerConfig {
	return SanitizerConfig{
		StripHTML:  true,
		EscapeHTML: true,
		MaxLength:  0, // No limit
	}
}

// NewSanitizer creates a new input sanitizer.
func NewSanitizer(cfg SanitizerConfig) *Sanitizer {
	return &Sanitizer{config: cfg}
}

// Sanitize sanitizes the input string.
func (s *Sanitizer) Sanitize(input string) string {
	result := input

	// Apply max length
	if s.config.MaxLength > 0 && len(result) > s.config.MaxLength {
		result = result[:s.config.MaxLength]
	}

	// Strip HTML tags if configured
	if s.config.StripHTML {
		result = stripHTMLTags(result)
	}

	// Escape HTML entities
	if s.config.EscapeHTML {
		result = html.EscapeString(result)
	}

	// Remove null bytes
	result = strings.ReplaceAll(result, "\x00", "")

	return result
}

// SanitizeMap sanitizes all string values in a map.
func (s *Sanitizer) SanitizeMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		result[k] = s.sanitizeValue(v)
	}
	return result
}

func (s *Sanitizer) sanitizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		return s.Sanitize(val)
	case map[string]interface{}:
		return s.SanitizeMap(val)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = s.sanitizeValue(item)
		}
		return result
	default:
		return v
	}
}

// HTML tag stripping regex
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func stripHTMLTags(input string) string {
	return htmlTagRegex.ReplaceAllString(input, "")
}

// =========================================================================
// SQL Injection Prevention
// =========================================================================

// SQLInjectionPatterns are common SQL injection patterns.
var SQLInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(\bor\b|\band\b).+(\b=\b|<|>)`),  // OR/AND conditions
	regexp.MustCompile(`(?i)\bunion\b.+\bselect\b`),          // UNION SELECT
	regexp.MustCompile(`(?i)\bselect\b.+\bfrom\b`),           // SELECT FROM
	regexp.MustCompile(`(?i)\binsert\b.+\binto\b`),           // INSERT INTO
	regexp.MustCompile(`(?i)\bdelete\b.+\bfrom\b`),           // DELETE FROM
	regexp.MustCompile(`(?i)\bdrop\b.+\b(table|database)\b`), // DROP TABLE/DATABASE
	regexp.MustCompile(`(?i)\bexec\b|\bexecute\b`),           // EXEC/EXECUTE
	regexp.MustCompile(`(?i)(--|#|/\*)`),                     // Comment markers
	regexp.MustCompile(`(?i)\bwaitfor\b.+\bdelay\b`),         // Time-based
	regexp.MustCompile(`(?i)\bsleep\b\s*\(`),                 // Sleep function
	regexp.MustCompile(`'.*(\bor\b|\band\b).*'`),             // Quote escaping
}

// DetectSQLInjection checks if input contains potential SQL injection.
// Returns true if suspicious patterns are detected.
func DetectSQLInjection(input string) bool {
	for _, pattern := range SQLInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// =========================================================================
// Path Traversal Prevention
// =========================================================================

// PathTraversalPatterns are common path traversal patterns.
var PathTraversalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.\.(/|\\)`),              // ../  or ..\
	regexp.MustCompile(`(/|\\)\.\.`),              // /..  or \..
	regexp.MustCompile(`%2e%2e(%2f|%5c)`),         // URL encoded ../
	regexp.MustCompile(`%252e%252e(%252f|%255c)`), // Double URL encoded
}

// DetectPathTraversal checks if input contains path traversal patterns.
func DetectPathTraversal(input string) bool {
	// Loop to handle multiple layers of encoding (e.g. %252e%252e%252f)
	// Limit to 5 iterations to prevent potential DoS or infinite loops
	for i := 0; i < 5; i++ {
		lower := strings.ToLower(input)
		for _, pattern := range PathTraversalPatterns {
			if pattern.MatchString(lower) {
				return true
			}
		}

		decoded, err := url.QueryUnescape(input)
		if err != nil || decoded == input {
			break
		}
		input = decoded
	}
	return false
}

// SanitizePath removes path traversal attempts from a path string.
func SanitizePath(input string) string {
	// Decode URL encoding to handle encoded traversal patterns (e.g. %2e%2e%2f)
	// We loop to handle multiple layers of encoding (e.g. %252e%252e%252f)
	// Limit to 5 iterations to prevent potential DoS or infinite loops
	for i := 0; i < 5; i++ {
		decoded, err := url.QueryUnescape(input)
		if err != nil || decoded == input {
			break
		}
		input = decoded
	}

	result := input
	for {
		original := result

		// Remove standard traversal patterns
		result = strings.ReplaceAll(result, "../", "")
		result = strings.ReplaceAll(result, "..\\", "")

		// Remove trailing traversal components
		result = strings.TrimSuffix(result, "/..")
		result = strings.TrimSuffix(result, "\\..")

		// Handle exact match ".."
		if result == ".." {
			result = ""
		}

		// If no changes, break
		if result == original {
			break
		}
	}
	return result
}

// =========================================================================
// Command Injection Prevention
// =========================================================================

// DangerousCharacters are characters that could enable command injection.
var DangerousCharacters = []string{
	";", "&", "|", "$", "`", "(", ")", "{", "}", "[", "]",
	"<", ">", "!", "\\", "'", "\"", "\n", "\r",
}

// DetectCommandInjection checks for potential command injection.
func DetectCommandInjection(input string) bool {
	for _, char := range DangerousCharacters {
		if strings.Contains(input, char) {
			return true
		}
	}
	return false
}

// SanitizeForShell removes or escapes dangerous characters for shell commands.
// WARNING: Prefer using parameterized commands instead of string interpolation.
func SanitizeForShell(input string) string {
	result := input
	for _, char := range DangerousCharacters {
		result = strings.ReplaceAll(result, char, "")
	}
	return result
}
