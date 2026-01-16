package jwt_test

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/jwt"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type JWTSuite struct {
	*test.Suite
}

func TestJWTSuite(t *testing.T) {
	test.Run(t, &JWTSuite{Suite: test.NewSuite()})
}

func (s *JWTSuite) TestTokenService() {
	secret := "test-secret-key-123"
	svc := jwt.New(jwt.Config{
		Secret:     secret,
		Expiration: time.Hour,
	})

	userID := "user-123"

	// 1. Generate Token
	token, err := svc.Generate(userID)
	s.NoError(err)
	s.NotEmpty(token)

	// 2. Validate Token
	claims, err := svc.Validate(token)
	s.NoError(err)
	s.Equal(userID, claims["sub"])

	// 3. Validate Invalid Token
	_, err = svc.Validate("invalid.token.string")
	s.Error(err)
}

func (s *JWTSuite) TestTokenService_Expired() {
	svc := jwt.New(jwt.Config{
		Secret:     "secret",
		Expiration: -1 * time.Hour, // Expired immediately
	})

	token, _ := svc.Generate("user-123")

	_, err := svc.Validate(token)
	s.Error(err)
	s.Contains(err.Error(), "token")
}
