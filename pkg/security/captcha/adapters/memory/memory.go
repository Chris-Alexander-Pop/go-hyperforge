package memory

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/captcha"
)

// Verifier implements captcha.Verifier using simple memory checks.
type Verifier struct {
	validToken string
}

// Ensure Verifier implements captcha.Verifier.
var _ captcha.Verifier = (*Verifier)(nil)

// New creates a new memory captcha verifier.
// It accepts a magic token that is considered valid. All others are invalid.
// Defaults to "valid-token" if empty.
func New(magicToken string) *Verifier {
	if magicToken == "" {
		magicToken = "valid-token"
	}
	return &Verifier{
		validToken: magicToken,
	}
}

func (v *Verifier) Verify(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == "" || token != v.validToken {
		return captcha.ErrInvalidToken
	}
	return nil
}
