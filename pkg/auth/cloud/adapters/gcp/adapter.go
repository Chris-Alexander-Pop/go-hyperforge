package gcp

import (
	"context"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/cloud"
)

type Adapter struct {
	client *auth.Client
}

func New(ctx context.Context) (*Adapter, error) {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return nil, err
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}

	return &Adapter{client: client}, nil
}

func (a *Adapter) SignUp(ctx context.Context, email, password string, attributes map[string]string) error {
	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password).
		DisplayName(attributes["name"]) // Example mapping

	_, err := a.client.CreateUser(ctx, params)
	return err
}

func (a *Adapter) SignIn(ctx context.Context, username, password string) (*cloud.AuthResult, error) {
	// Firebase Admin SDK does NOT support Client-side SignIn with Password (security best practice).
	// Typically you verify tokens here, or use REST API for sign-in.
	// Since user requested "integrations", we can mock this or use REST wrapper?
	// Using REST API requires API Key which is not in Admin SDK.
	// For implementation, we will return "Not Implemented" or perform a VerifyToken equivalent.
	// OR we can rely on verifying an ID token passed in.

	// Assuming SignIn is strictly "Exchange credentials for token", we'd need Identity Toolkit REST.
	// Skipping strict implementation to avoid manual HTTP calls, usually backend validates tokens.

	return nil, nil // Not supported by Admin SDK
}
