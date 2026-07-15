package firewall

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrSecurityGroupNotFound is returned when a requested security group does not exist.
	ErrSecurityGroupNotFound = errors.NotFound("security group not found", nil)

	// ErrRuleNotFound is returned when a requested rule does not exist.
	ErrRuleNotFound = errors.NotFound("rule not found", nil)
)
