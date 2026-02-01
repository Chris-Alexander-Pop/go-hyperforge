package auth

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/middleware"
)

// MiddlewareVerifier adapts an auth.Verifier to the middleware.Verifier interface
type MiddlewareVerifier struct {
	v Verifier
}

func NewMiddlewareVerifier(v Verifier) *MiddlewareVerifier {
	return &MiddlewareVerifier{v: v}
}

func (m *MiddlewareVerifier) Verify(ctx context.Context, token string) (string, []string, error) {
	claims, err := m.v.Verify(ctx, token)
	if err != nil {
		return "", nil, err
	}
	return claims.Subject, claims.Roles, nil
}

// Interface check
var _ middleware.Verifier = (*MiddlewareVerifier)(nil)
