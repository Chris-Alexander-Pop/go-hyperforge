package paseto

import (
	"context"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

type Config struct {
	SymmetricKey string `env:"PASETO_KEY" env-required:"true"` // Hex encoded 32 bytes
}

type Adapter struct {
	key paseto.V4SymmetricKey
}

func New(cfg Config) (*Adapter, error) {
	k, err := paseto.V4SymmetricKeyFromHex(cfg.SymmetricKey)
	if err != nil {
		return nil, errors.Wrap(err, "invalid paseto key hex")
	}
	return &Adapter{key: k}, nil
}

// Generate creates an encrypted V4 Local token
func (a *Adapter) Generate(userID string, role string, ttl time.Duration) (string, error) {
	token := paseto.NewToken()
	token.SetSubject(userID)
	token.Set("role", role)
	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	token.SetExpiration(time.Now().Add(ttl))

	return token.V4Encrypt(a.key, nil), nil
}

// Verify decrypts and validates the token
func (a *Adapter) Verify(ctx context.Context, tokenString string) (*auth.Claims, error) {
	parser := paseto.NewParser()
	// Add rules if strict logic needed

	token, err := parser.ParseV4Local(a.key, tokenString, nil)
	if err != nil {
		return nil, errors.New(errors.CodeUnauthenticated, "invalid paseto token", err)
	}

	// Extract Claims
	sub, _ := token.GetSubject()
	role, _ := token.GetString("role")
	iss, _ := token.GetIssuer()
	exp, _ := token.GetExpiration()
	iat, _ := token.GetIssuedAt()

	return &auth.Claims{
		Subject:   sub,
		Role:      role,
		Issuer:    iss,
		ExpiresAt: exp.Unix(),
		IssuedAt:  iat.Unix(),
	}, nil
}
