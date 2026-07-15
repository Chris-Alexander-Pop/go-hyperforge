package gcpidentity_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/gcpidentity"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewRequiresProjectID(t *testing.T) {
	_, err := gcpidentity.New(context.Background(), gcpidentity.Config{})
	if !auth.IsInvalidConfig(err) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}

func TestLoginRequiresAPIKey(t *testing.T) {
	// Firebase NewApp without credentials may fail in CI; skip if so.
	a, err := gcpidentity.New(context.Background(), gcpidentity.Config{
		ProjectID: "demo-project",
	})
	if err != nil {
		t.Skipf("skipping: firebase init unavailable: %v", err)
	}
	_, err = a.Login(context.Background(), "a@b.com", "pw")
	if !auth.IsInvalidConfig(err) {
		t.Fatalf("expected invalid config for missing API key, got %v", err)
	}
}

func TestLoginViaIdentityToolkitHTTP(t *testing.T) {
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.String(), "signInWithPassword") {
			t.Fatalf("unexpected URL: %s", r.URL)
		}
		body, _ := json.Marshal(map[string]string{
			"idToken":      "",
			"refreshToken": "refresh",
			"expiresIn":    "3600",
			"localId":      "uid-1",
			"email":        "a@b.com",
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(body))),
			Header:     make(http.Header),
		}, nil
	})

	a, err := gcpidentity.New(context.Background(), gcpidentity.Config{
		ProjectID: "demo-project",
		APIKey:    "test-key",
	}, gcpidentity.WithHTTPClient(&http.Client{Transport: rt}))
	if err != nil {
		t.Skipf("skipping: firebase init unavailable: %v", err)
	}

	claims, err := a.Login(context.Background(), "a@b.com", "pw")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if claims.Subject != "uid-1" || claims.Email != "a@b.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestLoginRejectsBadCredentialsFromAPI(t *testing.T) {
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"error":{"message":"INVALID_PASSWORD"}}`
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	a, err := gcpidentity.New(context.Background(), gcpidentity.Config{
		ProjectID: "demo-project",
		APIKey:    "test-key",
	}, gcpidentity.WithHTTPClient(&http.Client{Transport: rt}))
	if err != nil {
		t.Skipf("skipping: firebase init unavailable: %v", err)
	}

	_, err = a.Login(context.Background(), "a@b.com", "bad")
	if err == nil || !errors.IsCode(err, errors.CodeUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestVerifyEmptyToken(t *testing.T) {
	a, err := gcpidentity.New(context.Background(), gcpidentity.Config{ProjectID: "demo-project"})
	if err != nil {
		t.Skipf("skipping: firebase init unavailable: %v", err)
	}
	_, err = a.Verify(context.Background(), "")
	if !auth.IsInvalidToken(err) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}
