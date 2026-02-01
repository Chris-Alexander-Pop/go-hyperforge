package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Secret     string        `env:"JWT_SECRET" env-required:"true"`
	Expiration time.Duration `env:"JWT_EXPIRATION" env-default:"24h"`
	Issuer     string        `env:"JWT_ISSUER" env-default:"system-design-library"`
}

type Adapter struct {
	cfg Config
}

func New(cfg Config) *Adapter {
	return &Adapter{cfg: cfg}
}

// Verify implements auth.Verifier
func (a *Adapter) Verify(ctx context.Context, tokenString string) (*auth.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.InvalidArgument(fmt.Sprintf("unexpected signing method: %v", token.Header["alg"]), nil)
		}
		return []byte(a.cfg.Secret), nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "invalid token")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Map standard claims
		c := &auth.Claims{}
		if sub, ok := claims["sub"].(string); ok {
			c.Subject = sub
		}
		if iss, ok := claims["iss"].(string); ok {
			c.Issuer = iss
		}
		if role, ok := claims["role"].(string); ok {
			c.Roles = append(c.Roles, role)
		}
		if roles, ok := claims["roles"].([]interface{}); ok {
			for _, r := range roles {
				c.Roles = append(c.Roles, fmt.Sprintf("%v", r))
			}
		}

		return c, nil
	}

	return nil, errors.New(errors.CodeUnauthenticated, "invalid token claims", nil)
}

// Generate creates a new token (Specific to Local adapter)
func (a *Adapter) Generate(userID string, roles []string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"iss":   a.cfg.Issuer,
		"roles": roles,
		"exp":   time.Now().Add(a.cfg.Expiration).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.cfg.Secret))
}
