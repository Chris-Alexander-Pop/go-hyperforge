package cognito_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/cognito"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

func TestNewRequiresConfig(t *testing.T) {
	_, err := cognito.New(context.Background(), cognito.Config{})
	if !auth.IsInvalidConfig(err) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}

func TestVerifyEmptyToken(t *testing.T) {
	// Construct adapter without AWS by using incomplete path is hard;
	// Verify empty token does not need AWS once adapter exists.
	// We only assert the sentinel via a minimal adapter through New when AWS
	// credentials are absent — New may still succeed with default credential chain.
	a, err := cognito.New(context.Background(), cognito.Config{
		UserPoolID: "us-east-1_EXAMPLE",
		ClientID:   "client",
		Region:     "us-east-1",
	})
	if err != nil {
		t.Skipf("skipping: cannot create cognito adapter in this environment: %v", err)
	}

	_, err = a.Verify(context.Background(), "   ")
	if !auth.IsInvalidToken(err) && !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestLoginEmptyCredentials(t *testing.T) {
	a, err := cognito.New(context.Background(), cognito.Config{
		UserPoolID: "us-east-1_EXAMPLE",
		ClientID:   "client",
		Region:     "us-east-1",
	})
	if err != nil {
		t.Skipf("skipping: cannot create cognito adapter: %v", err)
	}
	_, err = a.Login(context.Background(), "", "x")
	if !auth.IsInvalidCredentials(err) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}
