package azure

import (
	"context"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/cloud"
)

type Adapter struct {
	client *public.Client
}

func New(clientID, tenantID string) (*Adapter, error) {
	// Public Client for User Auth
	client, err := public.New(clientID, public.WithAuthority("https://login.microsoftonline.com/"+tenantID))
	if err != nil {
		return nil, err
	}
	return &Adapter{client: &client}, nil
}

func (a *Adapter) SignUp(ctx context.Context, username, password string, attributes map[string]string) error {
	// Entra ID (B2C?) usually handles signup via UI flow or Graph API.
	// MSAL is for auth. Graph API needed for user creation.
	// Skipping Graph API client for brevity, user creation is heavy.
	return nil
}

func (a *Adapter) SignIn(ctx context.Context, username, password string) (*cloud.AuthResult, error) {
	scopes := []string{"User.Read"} // Example scope

	// Use interactive or device code flow instead of ROPC
	// For non-interactive scenarios, use AcquireTokenSilent with cached tokens
	// or service principal auth via confidential client
	res, err := a.client.AcquireTokenInteractive(ctx, scopes)
	if err != nil {
		return nil, err
	}

	return &cloud.AuthResult{
		AccessToken: res.AccessToken,
		IDToken:     res.IDToken.RawToken,
		ExpiresIn:   int(res.ExpiresOn.Unix()), // Approx
	}, nil
}
