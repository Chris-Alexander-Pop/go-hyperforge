package rbac

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Permission represents an Action on a Resource.
type Permission struct {
	Resource string // e.g. "users"
	Action   string // e.g. "read"
}

// Enforcer defines the RBAC contract.
type Enforcer interface {
	// Enforce checks if the subject (role) has permission to perform action on resource.
	Enforce(ctx context.Context, role string, resource string, action string) (bool, error)

	// AddPolicy adds a permission to a role.
	AddPolicy(role string, resource string, action string)
}

// SimpleEnforcer is an in-memory Enforcer guarded by SmartRWMutex.
type SimpleEnforcer struct {
	policies map[string]map[Permission]bool // role -> permission -> true
	mu       *concurrency.SmartRWMutex
}

func New() *SimpleEnforcer {
	return &SimpleEnforcer{
		policies: make(map[string]map[Permission]bool),
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "rbac-enforcer"}),
	}
}

func (e *SimpleEnforcer) AddPolicy(role string, resource string, action string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.policies[role]; !ok {
		e.policies[role] = make(map[Permission]bool)
	}
	e.policies[role][Permission{Resource: resource, Action: action}] = true
}

func (e *SimpleEnforcer) Enforce(ctx context.Context, role string, resource string, action string) (bool, error) {
	// Super admin check (no lock needed for constant roles).
	if role == "admin" || role == "superuser" {
		return true, nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	perms, ok := e.policies[role]
	if !ok {
		return false, nil
	}

	if perms[Permission{Resource: resource, Action: action}] {
		return true, nil
	}
	if perms[Permission{Resource: resource, Action: "*"}] {
		return true, nil
	}
	if perms[Permission{Resource: "*", Action: action}] {
		return true, nil
	}

	return false, nil
}
