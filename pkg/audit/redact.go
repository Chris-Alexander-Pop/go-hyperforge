package audit

import (
	"regexp"
	"strings"
)

// Redactor handles PII and sensitive data redaction.
type Redactor struct {
	patterns    []*redactionPattern
	replacement string
}

type redactionPattern struct {
	name    string
	pattern *regexp.Regexp
	mask    string
}

// RedactorConfig configures the redactor.
type RedactorConfig struct {
	// Replacement is the string to replace sensitive data with.
	Replacement string

	// CustomPatterns are additional patterns to redact.
	CustomPatterns map[string]string
}

// DefaultRedactorConfig returns sensible defaults.
func DefaultRedactorConfig() RedactorConfig {
	return RedactorConfig{
		Replacement: "[REDACTED]",
	}
}

var defaultCompiledPatterns = map[string]*regexp.Regexp{
	"credit_card":  regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
	"ssn":          regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
	"email":        regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`),
	"phone":        regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}\b`),
	"api_key":      regexp.MustCompile(`\b(?:sk|pk|api|key|token|secret)[_-]?[a-zA-Z0-9]{20,}\b`),
	"aws_key":      regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	"jwt":          regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*\b`),
	"ipv4":         regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
	"password_url": regexp.MustCompile(`(?i)(?:password|passwd|pwd|secret|token)=([^&\s]+)`),
}

// NewRedactor creates a new PII redactor.
// ⚡ Bolt: Pre-compile default regex patterns to avoid compilation overhead on instantiation
func NewRedactor(cfg RedactorConfig) *Redactor {
	if cfg.Replacement == "" {
		cfg.Replacement = "[REDACTED]"
	}

	r := &Redactor{
		replacement: cfg.Replacement,
		patterns:    make([]*redactionPattern, 0, len(defaultCompiledPatterns)+len(cfg.CustomPatterns)),
	}

	// Add pre-compiled default patterns
	for name, compiled := range defaultCompiledPatterns {
		r.patterns = append(r.patterns, &redactionPattern{
			name:    name,
			pattern: compiled,
			mask:    r.replacement,
		})
	}

	// Add custom patterns
	for name, pattern := range cfg.CustomPatterns {
		if err := r.AddPattern(name, pattern, ""); err != nil {
			// Skip patterns that fail to compile
			continue
		}
	}

	return r
}

// AddPattern adds a redaction pattern.
func (r *Redactor) AddPattern(name, pattern, mask string) error {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	if mask == "" {
		mask = r.replacement
	}

	r.patterns = append(r.patterns, &redactionPattern{
		name:    name,
		pattern: compiled,
		mask:    mask,
	})
	return nil
}

// Redact redacts sensitive data from a string.
func (r *Redactor) Redact(input string) string {
	result := input
	for _, p := range r.patterns {
		result = p.pattern.ReplaceAllString(result, p.mask)
	}
	return result
}

// RedactMap redacts sensitive data from a map.
func (r *Redactor) RedactMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		result[k] = r.redactValue(v)
	}
	return result
}

func (r *Redactor) redactValue(v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		return r.Redact(val)
	case map[string]interface{}:
		return r.RedactMap(val)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = r.redactValue(item)
		}
		return result
	default:
		return v
	}
}

// RedactEvent redacts sensitive data from an audit event.
func (r *Redactor) RedactEvent(event Event) Event {
	// Redact string fields
	event.ActorIP = r.Redact(event.ActorIP)
	event.ActorUserAgent = r.Redact(event.ActorUserAgent)
	event.Description = r.Redact(event.Description)
	event.ErrorMessage = r.Redact(event.ErrorMessage)

	// Redact metadata
	if event.Metadata != nil {
		event.Metadata = r.RedactMap(event.Metadata)
	}

	return event
}

// SensitiveFields returns a list of field names that should be redacted.
var SensitiveFields = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"api_key",
	"apikey",
	"access_token",
	"refresh_token",
	"authorization",
	"credit_card",
	"card_number",
	"cvv",
	"ssn",
	"social_security",
	"private_key",
}

// IsSensitiveField checks if a field name indicates sensitive data.
func IsSensitiveField(name string) bool {
	lower := strings.ToLower(name)
	for _, sensitive := range SensitiveFields {
		if strings.Contains(lower, sensitive) {
			return true
		}
	}
	return false
}

// MaskString partially masks a string, showing only the first/last few characters.
func MaskString(s string, showFirst, showLast int) string {
	if len(s) <= showFirst+showLast {
		return strings.Repeat("*", len(s))
	}
	return s[:showFirst] + strings.Repeat("*", len(s)-showFirst-showLast) + s[len(s)-showLast:]
}

// MaskEmail masks the local part of an email address.
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "[INVALID_EMAIL]"
	}
	local := parts[0]
	domain := parts[1]
	if len(local) <= 2 {
		return "*@" + domain
	}
	return local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:] + "@" + domain
}

// MaskCreditCard masks a credit card number, showing only the last 4 digits.
func MaskCreditCard(cc string) string {
	// Remove spaces and dashes
	cleaned := strings.ReplaceAll(strings.ReplaceAll(cc, " ", ""), "-", "")
	if len(cleaned) < 4 {
		return "[INVALID_CC]"
	}
	return strings.Repeat("*", len(cleaned)-4) + cleaned[len(cleaned)-4:]
}
