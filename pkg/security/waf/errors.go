package waf

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

const (
	CodeInvalidRule = "WAF_INVALID_RULE"
	CodeNotFound    = "WAF_NOT_FOUND"
	CodeUnavailable = "WAF_UNAVAILABLE"
)

var (
	// ErrInvalidRule is returned when a WAF rule or IP is malformed.
	ErrInvalidRule = errors.New(CodeInvalidRule, "invalid waf rule", nil)

	// ErrNotFound is returned when a rule or IP entry does not exist.
	ErrNotFound = errors.New(CodeNotFound, "waf rule not found", nil)

	// ErrUnavailable is returned when a remote WAF API is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "waf provider unavailable", nil)
)
