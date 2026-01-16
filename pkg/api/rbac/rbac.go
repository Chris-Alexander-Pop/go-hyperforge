package rbac

import (
	"context"
)

// Permission represents an Action on a Resource
type Permission struct {
	Resource string // e.g. "users"
	Action   string // e.g. "read"
}

// Enforcer defines the RBAC contract
type Enforcer interface {
	// Enforce checks if the subject (role) has permission to perform action on resource
	Enforce(ctx context.Context, role string, resource string, action string) (bool, error)

	// AddPolicy adds a permission to a role
	AddPolicy(role string, resource string, action string)
}

// SimpleEnforcer in-memory implementation
type SimpleEnforcer struct {
	policies map[string]map[Permission]bool // role -> permission -> true
}

func New() *SimpleEnforcer {
	return &SimpleEnforcer{
		policies: make(map[string]map[Permission]bool),
	}
}

func (e *SimpleEnforcer) AddPolicy(role string, resource string, action string) {
	if _, ok := e.policies[role]; !ok {
		e.policies[role] = make(map[Permission]bool)
	}
	e.policies[role][Permission{Resource: resource, Action: action}] = true
}

func (e *SimpleEnforcer) Enforce(ctx context.Context, role string, resource string, action string) (bool, error) {
	// Super admin check
	if role == "admin" || role == "superuser" {
		return true, nil
	}

	perms, ok := e.policies[role]
	if !ok {
		return false, nil
	}

	// Exact match
	if perms[Permission{Resource: resource, Action: action}] {
		return true, nil
	}

	// Wildcard Action match (e.g. users:*)
	if perms[Permission{Resource: resource, Action: "*"}] {
		return true, nil
	}

	// Wildcard Resource match (e.g. *:read) - rare but possible
	if perms[Permission{Resource: "*", Action: action}] {
		return true, nil
	}

	return false, nil
}
