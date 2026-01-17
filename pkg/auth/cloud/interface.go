package cloud

import (
	"context"
)

// IdentityProvider abstracts cloud identity services.
type IdentityProvider interface {
	// SignUp registers a new user.
	SignUp(ctx context.Context, username, password string, attributes map[string]string) error

	// SignIn authenticates a user and returns result (e.g., token).
	SignIn(ctx context.Context, username, password string) (*AuthResult, error)
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	ExpiresIn    int
}
